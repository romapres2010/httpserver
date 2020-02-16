Mini
================

[![Build Status](https://travis-ci.org/sasbury/mini.svg?branch=master)](https://travis-ci.org/sasbury/mini) [![GoDoc](https://godoc.org/github.com/sasbury/mini?status.svg)](https://godoc.org/github.com/sasbury/mini)

Mini is a simple [ini configuration file](http://en.wikipedia.org/wiki/INI_file) parser.

The ini syntax supported includes:

* The standard name=value
* Comments on new lines starting with # or ;
* Blank lines
* Sections labelled with [sectionname]
* Split sections, using the same section in more than one place
* Encoded strings, strings containing \n, \t, etc...
* Array values using repeated keys named in the form key[]=value
* Global key/value pairs that appear before the first section

Repeated keys, that aren't array keys, replace their previous value.

To use simply:

    % go get github.com/sasbury/mini
