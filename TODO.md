General non-specific stuff ill do

- ### api
    - translate (each provider has their own path)
    - login (with refresh paths)
    - healthcheck
    - restricted registeration
- ### auth
    - password with agron2id encryption
    - JWT for sessions
    - probably wont need refresh tokens as this is a per sesion use with api costs
- ### cache
    - use redis for cacheing
    - cache api requests for a set duration
- ### database
    - use postgresql
    - store users and passwords
- ### login
    - admin user (usually the server owner) can create new users if needed
- ### make a docker compose for easy server setup

possible stuff that i may do:

- ### logging with Prometheus
    - log all metrics needed
    - use graphana for visualization


### api endpoints by importance:

- /v1/deepl/translate
- /v1/deepl/documents (GET /id = check status or get result, DELETE /id = delete document)
- /v1/auth/login
- /v1/admin/users (POST = create , GET = get users , DELETE = delete user, GET /id query id)
- /v1/healthcheck

