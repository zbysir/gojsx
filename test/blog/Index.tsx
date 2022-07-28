import Header from "./Header";
import Home from "./page/Home";
import BlogDetail from "./page/BlogDetail";

interface Props {
    page: 'home' | 'blog-detail'
    title: string
    pageData: any
    me: string
    time: string
}

export default function Index(props: Props) {
    return <html lang="zh">
    <head>
        <meta charSet="UTF-8"/>
        <title>{props.title || 'UnTitled'}</title>
        <link href="/tailwind.css" rel="stylesheet"/>
    </head>
    <body>
    <Header name={props.me}></Header>
    {
        (function () {
            switch (props.page) {
                case 'home':
                    return <Home {...props.pageData}></Home>
                case 'blog-detail':
                    return <BlogDetail {...props.pageData}></BlogDetail>
            }
            return props.page
        })()
    }
    <div>
        {props.time}
    </div>
    </body>
    </html>
}
