
- ### metrics with Prometheus
    - log all metrics needed
    - use graphana for visualization
- ### async document translation
    - DONE POST /translate returns {document_id}
    - GET /documents/{id} returns status
    - GET /documents/{id}/download returns file
    - DONE store metadata in postgres
    - store binary in redis with TTL

- ### custom error package
    - DONE Refactor all errors to use a the custom error package
    - DONE the error package will be in the mass-translate-package 
    - DONE use fmt.Errorf("%s | %w : %s",package,err,x) format with error constants in the custom err package
    - DONE store error within logging in PostgreSQL

- ### Clients
    - create a simple CLI tool to query the server 
    - make the a website frontend for the server with login using typescript (this is much later)

### Low Priority

- Rename PostgreSQL database from `masstranslate` to `sublate`
### api endpoints to be done:

- /v1/documents (GET /id = check status or get result, DELETE /id = delete document)
- /v1/admin/logs  (GET?n=10 get top n logs , GET /{id} query log id)
- /v1/admin/metrics

