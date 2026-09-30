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

// Create a new predictor instance.
// model_dir must contain model.bin and tokenizer.model.
REPAROS_API ReparosHandle reparos_create(
    const char* model_dir,
    const char* device,
    const char* compute_type
);

// Predict top corrected query hypotheses.
// Returns 0 on success, non-zero on failure.
REPAROS_API int reparos_predict(
    ReparosHandle handle,
    const char* query,
    int beam_size,
    int num_hypotheses,
    char*** out_hypotheses,
    float** out_scores,
    int* out_count
);

// Free hypotheses and scores allocated by reparos_predict.
REPAROS_API void reparos_free_results(
    char** hypotheses,
    float* scores,
    int count
);

// Destroy the predictor instance.
REPAROS_API void reparos_free(ReparosHandle handle);

#ifdef __cplusplus
}
#endif

#endif // REPAROS_BRIDGE_H
