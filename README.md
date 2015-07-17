# pollbc
Poll leboncoin.fr and notify when new announces are available

See it running at https://pollbc.herokuapp.com/

## Run
To develop locally, a running instance of PostgreSQL is required. On OSX, type:

    brew install postgres # then follow instructions to start postgresql
    createdb

Then add `DATABASE_URL=sslmode=disable` to your env and run with:

    go install && foreman start
