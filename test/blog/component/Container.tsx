interface Props {
    children?: any[]
}

export default function Container(props: Props) {
    return <div className="container mx-auto p-6 bg-white flex">
        {props.children}
    </div>
}
