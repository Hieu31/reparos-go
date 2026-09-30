import sys
import json
import time
from pathlib import Path

# Ensure stdout uses UTF-8 without buffering
if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', line_buffering=True)
if hasattr(sys.stdin, 'reconfigure'):
    sys.stdin.reconfigure(encoding='utf-8')

def main():
    if len(sys.argv) < 2:
        sys.stderr.write("Usage: python runner.py <model_dir> [device] [compute_type]\n")
        sys.exit(1)

    model_dir = sys.argv[1]
    device = sys.argv[2] if len(sys.argv) > 2 else "cpu"
    compute_type = sys.argv[3] if len(sys.argv) > 3 else "int8"

    import ctranslate2
    import sentencepiece as spm

    tokenizer_path = str(Path(model_dir) / "tokenizer.model")
    sp = spm.SentencePieceProcessor()
    if not sp.load(tokenizer_path):
        sys.stderr.write(f"Failed to load tokenizer from {tokenizer_path}\n")
        sys.exit(1)

    translator = ctranslate2.Translator(model_dir, device=device, compute_type=compute_type)

    # Warmup OpenMP thread pool and CPU cache
    warm_pieces = sp.encode("khoi dong", out_type=str)
    translator.translate_batch([warm_pieces], beam_size=2)

    # Signal ready
    sys.stdout.write("READY\n")
    sys.stdout.flush()

    for line in sys.stdin:
        line = line.strip()
        if not line or line == "QUIT":
            break

        try:
            req = json.loads(line)
            query = req.get("query", "")
            beam_size = req.get("beam_size", 10)
            num_hypotheses = req.get("num_hypotheses", 1)

            t0 = time.perf_counter()
            pieces = sp.encode(query, out_type=str)
            max_len = min(max(len(pieces) * 2 + 6, 12), 60)
            res = translator.translate_batch(
                [pieces],
                beam_size=beam_size,
                num_hypotheses=num_hypotheses,
                max_decoding_length=max_len,
                return_scores=True
            )[0]

            hypotheses = [sp.decode(h) for h in res.hypotheses]
            scores = list(res.scores)
            latency = (time.perf_counter() - t0) * 1000

            resp = {
                "top1_query": hypotheses[0] if hypotheses else query,
                "hypotheses": hypotheses,
                "scores": scores,
                "latency_ms": latency,
                "changed": hypotheses[0] != query if hypotheses else False
            }
            sys.stdout.write(json.dumps(resp, ensure_ascii=False) + "\n")
            sys.stdout.flush()
        except Exception as e:
            err = {"error": str(e)}
            sys.stdout.write(json.dumps(err) + "\n")
            sys.stdout.flush()

if __name__ == "__main__":
    main()
