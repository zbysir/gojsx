const htmr  = require( 'htmr');
// import React from 'react';
const { renderToString }  = require( 'react-dom/server')
const s = renderToString(htmr(`
<>
  {1}
<>
<div>?<div>
<Toc items = {toc}></Toc>`))
console.log(s)