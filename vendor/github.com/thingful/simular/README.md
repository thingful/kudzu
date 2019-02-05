# simular [![Build Status](https://travis-ci.org/thingful/simular.png?branch=master)](https://travis-ci.org/thingful/simular) [![GoDoc](https://godoc.org/github.com/thingful/simular?status.svg)](https://godoc.org/github.com/thingful/simular)

This library is a fork of https://github.com/jarcoal/httpmock, which now has a
somewhat different API from the original library. All of the clever stuff was
worked out in the original version, this fork just alters the API for creating
stubbed requests, adds some extra fussiness in how it registers and responds to
requests, and adds an API for verifying that all stubbed requests were actually
called.

Originally this library was forked here: https://github.com/thingful/httpmock,
but the API here has changed sufficiently that we decided to just create a
completely new project, which has been renamed as `simular`.

## What is the purpose of this fork?

The reason for creating this fork was that while the original httpmock library
provided a very neat mechanism for inserting mocked responses into the default
Go net/http client, it was intentionally designed to be very tolerant about the
requests it matched which lead to us seeing some bugs in production.

To give a specific example let's say our code requires requesting data from
some external resource (http://api.example.com/resource/1), and for correct
operation it must include in that request a specific authorization header
containing a value passed in from somewhere else within the program. For this
scenario the previous library would let us create a responder for the resource
http://api.example.com/resource/1, but wouldn't allow us to force a test
failure if that resource was requested without the authorization header being
present.

## Usage examples

For usage examples, please see the documentation at
https://godoc.org/github.com/thingful/simular.

## Alternatives

* [httpmock](https://github.com/jarcoal/httpmock) - the original library this
  code was based on.
* [gock](https://github.com/h2non/gock) - Gock does everything this library does
  and more, and if it had existed when this codebase was first created I
  probably would have just used that instead.
* [govcr](https://github.com/seborama/govcr) - port of the Ruby VCR gem. Allows
  recording of requests and responses which can then be played back within
  tests.
