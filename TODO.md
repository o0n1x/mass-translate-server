
- ### metrics with Prometheus
    - log all metrics needed
    - use graphana for visualization
- ### async document translation
    - POST /translate returns {document_id, status: pending}
    - GET /documents/{id} returns status
    - GET /documents/{id}/download returns file
    - store metadata in postgres
    - store binary in redis with TTL

- ### custom error package
    - DONE Refactor all errors to use a the custom error package
    - DONE the error package will be in the mass-translate-package 
    - DONE use fmt.Errorf("%s | %w : %s",package,err,x) format with error constants in the custom err package
    - store error within logging in PostgreSQL

### Low Priority

- Rename PostgreSQL database from `masstranslate` to `sublate`
### api endpoints to be done:

- /v1/deepl/documents (GET /id = check status or get result, DELETE /id = delete document)
- /v1/admin/logs  (GET?n=10 get top n logs , GET /{id} query log id)
- /v1/admin/metrics

