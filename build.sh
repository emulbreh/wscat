#!/bin/bash

export GO15VENDOREXPERIMENT=1
glide install
go install

