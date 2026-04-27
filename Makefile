#!make
ifneq (,$(wildcard ./.env))
    include ./.env
    export $(shell sed 's/=.*//' ./.env)
endif

include scripts/system/*.mk
include scripts/usr/*.mk
