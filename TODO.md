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