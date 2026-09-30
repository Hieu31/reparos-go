#include "bridge.h"

#include <ctranslate2/translator.h>
#include <sentencepiece_processor.h>

#include <cstring>
#include <memory>
#include <string>
#include <vector>

struct ReparosEngine {
    std::unique_ptr<ctranslate2::Translator> translator;
    std::unique_ptr<sentencepiece::SentencePieceProcessor> tokenizer;
};

static char* duplicate_string(const std::string& str) {
    char* copy = new char[str.size() + 1];
    std::memcpy(copy, str.c_str(), str.size() + 1);
    return copy;
}

extern "C" {

ReparosHandle reparos_create(
    const char* model_dir,
    const char* device,
    const char* compute_type
) {
    if (!model_dir) return nullptr;

    try {
        auto engine = std::make_unique<ReparosEngine>();

        // 1. Initialize SentencePiece tokenizer
        std::string tokenizer_path = std::string(model_dir) + "/tokenizer.model";
        engine->tokenizer = std::make_unique<sentencepiece::SentencePieceProcessor>();
        auto status = engine->tokenizer->Load(tokenizer_path);
        if (!status.ok()) {
            return nullptr;
        }

        // 2. Parse device and compute type
        ctranslate2::Device dev = ctranslate2::Device::CPU;
        if (device && std::string(device) == "cuda") {
            dev = ctranslate2::Device::CUDA;
        }

        ctranslate2::ComputeType ct = ctranslate2::ComputeType::DEFAULT;
        if (compute_type) {
            std::string ct_str = compute_type;
            if (ct_str == "int8") ct = ctranslate2::ComputeType::INT8;
            else if (ct_str == "float16") ct = ctranslate2::ComputeType::FLOAT16;
            else if (ct_str == "float32") ct = ctranslate2::ComputeType::FLOAT32;
        }

        // 3. Initialize CTranslate2 Translator
        engine->translator = std::make_unique<ctranslate2::Translator>(
            model_dir,
            dev,
            ct
        );

        return static_cast<ReparosHandle>(engine.release());
    } catch (...) {
        return nullptr;
    }
}

int reparos_predict(
    ReparosHandle handle,
    const char* query,
    int beam_size,
    int num_hypotheses,
    char*** out_hypotheses,
    float** out_scores,
    int* out_count
) {
    if (!handle || !query || !out_hypotheses || !out_scores || !out_count) {
        return -1;
    }

    auto* engine = static_cast<ReparosEngine*>(handle);

    try {
        // 1. Tokenize query
        std::vector<std::string> tokens;
        auto status = engine->tokenizer->Encode(query, &tokens);
        if (!status.ok()) {
            return -2;
        }

        // 2. Set translation options
        ctranslate2::TranslationOptions opts;
        opts.beam_size = beam_size > 0 ? beam_size : 10;
        opts.num_hypotheses = num_hypotheses > 0 ? num_hypotheses : 1;
        opts.return_scores = true;
        opts.max_decoding_length = 100;

        // 3. Run CTranslate2 inference
        std::vector<std::vector<std::string>> batch = {tokens};
        auto results = engine->translator->translate_batch(batch, opts);
        if (results.empty()) {
            return -3;
        }

        const auto& result = results[0];
        size_t count = result.hypotheses.size();

        char** hypotheses = new char*[count];
        float* scores = new float[count];

        for (size_t i = 0; i < count; ++i) {
            std::string detokenized;
            engine->tokenizer->Decode(result.hypotheses[i], &detokenized);
            hypotheses[i] = duplicate_string(detokenized);
            scores[i] = (i < result.scores.size()) ? result.scores[i] : 0.0f;
        }

        *out_hypotheses = hypotheses;
        *out_scores = scores;
        *out_count = static_cast<int>(count);
        return 0;
    } catch (...) {
        return -4;
    }
}

void reparos_free_results(
    char** hypotheses,
    float* scores,
    int count
) {
    if (hypotheses) {
        for (int i = 0; i < count; ++i) {
            delete[] hypotheses[i];
        }
        delete[] hypotheses;
    }
    delete[] scores;
}

void reparos_free(ReparosHandle handle) {
    if (handle) {
        delete static_cast<ReparosEngine*>(handle);
    }
}

} // extern "C"
