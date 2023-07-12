export default function H({js,children}) {
    return <html>
    <head>
        <title>test</title>
    </head>
    <body>
    {children}

    <script type={"module"} dangerouslySetInnerHTML={{__html: js}}></script>
    </body>
    </html>
}