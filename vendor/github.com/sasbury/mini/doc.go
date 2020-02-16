/*
Package mini implements a simple ini file parser.

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

copyright © 2015 Fog Creek Software, Inc.
*/
package mini
