General non-specific stuff ill do

- ### api
    - DONE translate (each provider has their own path)
    - login (with refresh paths)
    - DONE healthcheck
    - restricted registeration
- ### auth
    - password with agron2id encryption
    - JWT for sessions
    - probably wont need refresh tokens as this is a per sesion use with api costs
- ### cache
    - DONE use redis for cacheing
    - DONE cache api requests for a set duration
- ### database
    - DONE use postgresql
    - store users and passwords
- ### login
    - admin user (usually the server owner) can create new users if needed
- ### make a docker compose for easy server setup

possible stuff that i may do:

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
    - Refactor all errors to use a the custom error package
    - the error package will be in the mass-translate-package 
    - use fmt.Errorf("%s | %w : %s",package,err,x) format with error constants in the custom err package
    - store error within logging in PostgreSQL

### api endpoints by importance:

- DONE /v1/deepl/translate  
- /v1/deepl/documents (GET /id = check status or get result, DELETE /id = delete document)
- /v1/auth/login
- /v1/admin/users (POST = create , GET = get users , DELETE = delete user, GET /id query id)
- /v1/admin/logs  (GET?n=10 get top n logs , GET /{id} query log id)
- DONE /v1/healthcheck

