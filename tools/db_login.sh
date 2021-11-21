#!/bin/sh

psql "host=katya-api.sandyuraz.com port=5432 user=sandy dbname=sandy sslmode=verify-full sslcert=./client/client.crt sslkey=./client/client.key sslrootcert=./ca/ca.crt"

