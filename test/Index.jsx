import App from "./App";

export default function Index(props) {
  return <html lang="zh">
  <head>
    <meta charSet="UTF-8"/>
    <title>{props.title || 'UnTitled'}</title>
    <link href="https://unpkg.com/tailwindcss@^2/dist/tailwind.min.css" rel="stylesheet"/>
  </head>
  <body>
  <App {...props}></App>
  </body>
  </html>
}
