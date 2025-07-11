# Demo HTTP Cache Server

This is just a demo. It listens only on localhost and uses a temporary directory
for storage, which it tries to delete before exiting. Do not run it except for
testing.

By default it listens on port 51918 (`0xcace`), but you can change this with the
`PORT` environment variable. You cannot change it from listening only on
localhost.
