#include "bridge.h"

#include <ctranslate2/translator.h>
#include <sentencepiece_processor.h>

#include <algorithm>
#include <cstring>
#include <memory>
#include <string>
#include <vector>

namespace {

constexpr int kAbiVersion = 2;
constexpr size_t kMaxQueryBytes = 1024;
constexpr size_t kMaxQueryTokens = 128;

struct Engine {
    std::unique_ptr<ctranslate2::Translator> translator;
    std::unique_ptr<sentencepiece::SentencePieceProcessor> tokenizer;
};

struct Result {
    std::vector<std::string> texts;
    std::vector<float> scores;
};

}  // namespace

extern "C" {

int reparos_abi_version(void) { return kAbiVersion; }

int reparos_create(
    const char* model_dir,
    const char* device,
    const char* compute_type,
    int intra_threads,
    int inter_threads,
    ReparosHandle* out
) {
    if (!model_dir || !out) return REPAROS_ERR_ARGUMENT;
    *out = nullptr;

    try {
        auto engine = std::make_unique<Engine>();

        engine->tokenizer = std::make_unique<sentencepiece::SentencePieceProcessor>();
        const auto status = engine->tokenizer->Load(std::string(model_dir) + "/tokenizer.model");
        if (!status.ok()) return REPAROS_ERR_LOAD;

        ctranslate2::Device dev = ctranslate2::Device::CPU;
        if (device && std::string(device) == "cuda") dev = ctranslate2::Device::CUDA;

        ctranslate2::ComputeType ct = ctranslate2::ComputeType::DEFAULT;
        if (compute_type) {
            const std::string name = compute_type;
            if (name == "int8") ct = ctranslate2::ComputeType::INT8;
            else if (name == "float16") ct = ctranslate2::ComputeType::FLOAT16;
            else if (name == "float32") ct = ctranslate2::ComputeType::FLOAT32;
        }

        ctranslate2::ReplicaPoolConfig pool;
        pool.num_threads_per_replica = intra_threads > 0 ? static_cast<size_t>(intra_threads) : 0;

        const size_t replicas = inter_threads > 0 ? static_cast<size_t>(inter_threads) : 1;
        engine->translator = std::make_unique<ctranslate2::Translator>(
            model_dir, dev, ct, std::vector<int>(replicas, 0), /*tensor_parallel=*/false, pool);

        // Warm up thread pool and CPU caches so the first request is not slow.
        std::vector<std::string> pieces;
        if (engine->tokenizer->Encode("khoi dong", &pieces).ok()) {
            ctranslate2::TranslationOptions warm;
            warm.beam_size = 2;
            warm.max_decoding_length = 12;
            engine->translator->translate_batch({pieces}, warm);
        }

        *out = engine.release();
        return REPAROS_OK;
    } catch (...) {
        return REPAROS_ERR_LOAD;
    }
}

int reparos_predict(
    ReparosHandle handle,
    const char* query,
    int beam_size,
    int num_hypotheses,
    ReparosResult* out
) {
    if (!handle || !query || !out) return REPAROS_ERR_ARGUMENT;
    *out = nullptr;

    auto* engine = static_cast<Engine*>(handle);

    try {
        if (std::strlen(query) > kMaxQueryBytes) return REPAROS_ERR_INPUT_TOO_LONG;

        std::vector<std::string> tokens;
        if (!engine->tokenizer->Encode(query, &tokens).ok()) return REPAROS_ERR_TOKENIZE;
        if (tokens.size() > kMaxQueryTokens) return REPAROS_ERR_INPUT_TOO_LONG;

        ctranslate2::TranslationOptions opts;
        opts.beam_size = beam_size > 0 ? static_cast<size_t>(beam_size) : 10;
        opts.num_hypotheses = num_hypotheses > 0 ? static_cast<size_t>(num_hypotheses) : 1;
        opts.return_scores = true;
        // Same bound as the reference Python runner: min(max(2n + 6, 12), 60).
        opts.max_decoding_length = std::min<size_t>(std::max<size_t>(tokens.size() * 2 + 6, 12), 60);

        auto results = engine->translator->translate_batch({tokens}, opts);
        if (results.empty()) return REPAROS_ERR_TRANSLATE;

        const auto& best = results[0];
        auto result = std::make_unique<Result>();
        result->texts.reserve(best.hypotheses.size());
        for (size_t i = 0; i < best.hypotheses.size(); ++i) {
            std::string text;
            if (!engine->tokenizer->Decode(best.hypotheses[i], &text).ok()) {
                return REPAROS_ERR_TOKENIZE;
            }
            result->texts.push_back(std::move(text));
            result->scores.push_back(i < best.scores.size() ? best.scores[i] : 0.0f);
        }

        *out = result.release();
        return REPAROS_OK;
    } catch (...) {
        return REPAROS_ERR_INTERNAL;
    }
}

int reparos_result_count(ReparosResult result) {
    if (!result) return 0;
    return static_cast<int>(static_cast<Result*>(result)->texts.size());
}

int reparos_result_scores(ReparosResult result, float* out, int cap) {
    if (!result) return 0;
    const auto* r = static_cast<Result*>(result);
    const int n = static_cast<int>(r->scores.size());
    if (out) {
        for (int i = 0; i < n && i < cap; ++i) out[i] = r->scores[i];
    }
    return n;
}

int reparos_result_text(ReparosResult result, int index, char* buf, int cap) {
    if (!result) return REPAROS_ERR_ARGUMENT;
    const auto* r = static_cast<Result*>(result);
    if (index < 0 || static_cast<size_t>(index) >= r->texts.size()) return REPAROS_ERR_ARGUMENT;

    const std::string& text = r->texts[index];
    if (buf && cap > 0) {
        const size_t n = std::min(text.size(), static_cast<size_t>(cap) - 1);
        std::memcpy(buf, text.data(), n);
        buf[n] = '\0';
    }
    return static_cast<int>(text.size());
}

void reparos_result_free(ReparosResult result) {
    delete static_cast<Result*>(result);
}

void reparos_free(ReparosHandle handle) {
    delete static_cast<Engine*>(handle);
}

}  // extern "C"
