const htmr  = require( 'htmr');
// import React from 'react';
const { renderToString }  = require( 'react-dom/server')
const s = renderToString(htmr(`<></>`))
console.log(s)

