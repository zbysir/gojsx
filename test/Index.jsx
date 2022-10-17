import App from "./App";
// import {useEffect, useState} from "react";

export default function Index(props) {
  // useEffect(() => {
  // }, []);
  return <html lang="zh">
  <head>
    <meta charSet="UTF-8"/>
    <title>{props.title || 'UnTitled'}</title>
    <link href="https://unpkg.com/tailwindcss@^2/dist/tailwind.min.css" rel="stylesheet"/>
  </head>
  <body>
  <div a={()=>{}} b={1} c={1.1}></div>
  <App {...props}

  ></App>
  </body>
  </html>
}
