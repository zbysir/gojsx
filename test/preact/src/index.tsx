export default function H({js,children}) {
    return <html>
    <head>
        <title>test</title>
    </head>
    <body>
    {children}

    {/*<style dangerouslySetInnerHTML={{__html: `[data-component-type=col]{padding: 20px}`}}></style>*/}

    <script type={"module"} dangerouslySetInnerHTML={{__html: js}}></script>
    </body>
    </html>
}