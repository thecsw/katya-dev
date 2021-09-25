#!/bin/sh

psql "host=katya.sandyuraz.com port=5432 user=sandy dbname=katya sslmode=verify-full sslcert=./client/client.crt sslkey=./client/client.key sslrootcert=./ca/ca.crt"

