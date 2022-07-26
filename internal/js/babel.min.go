package js

import _ "embed"

// BabelSrc copy from https://babel.docschina.org/docs/en/babel-standalone/
//go:embed babel.min.js
var Babel string

//go:embed jsx.js
var Jsx []byte
