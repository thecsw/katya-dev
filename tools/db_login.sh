#!/bin/sh

psql "host=elephant.sandyuraz.com port=5432 user=sandy dbname=sandissa sslmode=verify-full sslcert=./client/client.crt sslkey=./client/client.key sslrootcert=./ca/ca.crt"

