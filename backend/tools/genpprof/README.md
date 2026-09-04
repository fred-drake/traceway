# genpprof

Generates a synthetic CPU pprof for manually exercising the profiling label
pipeline (allowlist + label-separated dedup). The profile carries three CPU
samples on a single stack: two for `endpoint=/checkout` (100 + 25) and one for
`endpoint=/cart` (50), each also tagged with a high-cardinality `request_id`.

After ingest the decoder should keep only the allowlisted `endpoint` label,
drop `request_id`, and emit two samples (`/checkout`=125, `/cart`=50) sharing
one stack row.

## Generate

```bash
cd backend
go run ./tools/genpprof -out labeled.pprof          # raw pprof
go run ./tools/genpprof -out labeled.pprof.gz -gzip # gzipped, ready to POST
```

## End-to-end against a running backend

```bash
# backend on :8082 (DB_TYPE=sqlite is the simplest local path)
go run ./tools/genpprof -out labeled.pprof.gz -gzip

curl -sS -X POST "http://localhost:8082/profiles/ingest?service=checkout-svc&serverName=pod-a&appVersion=9.9.9" \
  -H "Authorization: Bearer <PROJECT_TOKEN>" \
  -H "Content-Encoding: gzip" \
  --data-binary @labeled.pprof.gz

curl -sS -X POST "http://localhost:8082/profiles/flamegraph?projectId=<PROJECT_ID>" \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"serviceName":"checkout-svc","type":"go:profile_cpu:nanoseconds","labels":{"endpoint":"/checkout"}}'
```

The `/checkout`-filtered flame graph should weigh 125; dropping the `labels`
filter weighs 175 (both endpoints); a `request_id` filter matches nothing
because that key is never stored.
