#ifndef REPAROS_BRIDGE_H
#define REPAROS_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

#if defined(_WIN32) || defined(_WIN64)
  #ifdef REPAROS_EXPORTS
    #define REPAROS_API __declspec(dllexport)
  #else
    #define REPAROS_API __declspec(dllimport)
  #endif
#else
  #define REPAROS_API __attribute__((visibility("default")))
#endif

typedef void* ReparosHandle;
typedef void* ReparosResult;

// Status codes returned by the API. 0 means success.
#define REPAROS_OK                 0
#define REPAROS_ERR_ARGUMENT      -1
#define REPAROS_ERR_TOKENIZE      -2
#define REPAROS_ERR_TRANSLATE     -3
#define REPAROS_ERR_INTERNAL      -4
#define REPAROS_ERR_INPUT_TOO_LONG -5
#define REPAROS_ERR_LOAD          -6

// Create a predictor. model_dir must contain model.bin and tokenizer.model.
// intra_threads: threads per translation (0 = library default).
// inter_threads: translations that may run in parallel (<= 0 means 1).
// The model is warmed up before returning. *out receives the handle.
REPAROS_API int reparos_create(
    const char* model_dir,
    const char* device,
    const char* compute_type,
    int intra_threads,
    int inter_threads,
    ReparosHandle* out
);

// Predict corrected hypotheses. Thread-safe on a shared handle.
// *out receives a result that must be released with reparos_result_free.
REPAROS_API int reparos_predict(
    ReparosHandle handle,
    const char* query,
    int beam_size,
    int num_hypotheses,
    ReparosResult* out
);

REPAROS_API int reparos_result_count(ReparosResult result);
// Copy up to cap scores into out. Returns the number of hypotheses.
REPAROS_API int reparos_result_scores(ReparosResult result, float* out, int cap);

// Copy hypothesis `index` into buf (NUL-terminated, truncated to cap).
// Returns the full length in bytes (excluding NUL), or a negative status.
// Call with cap = 0 to query the required size.
REPAROS_API int reparos_result_text(ReparosResult result, int index, char* buf, int cap);

REPAROS_API void reparos_result_free(ReparosResult result);
REPAROS_API void reparos_free(ReparosHandle handle);

// ABI version, bumped on incompatible changes.
REPAROS_API int reparos_abi_version(void);

#ifdef __cplusplus
}
#endif

#endif // REPAROS_BRIDGE_H
