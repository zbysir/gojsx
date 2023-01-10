interface Item {
    time: string
    content: string
}

interface Props {
    items: Item[]
}

export default ({items}: Props) => {
    return <div className={'flex nowrap'}>
        {items.map(i =>
            <div className={"flex col p-4"}>
                <p className={""}>{i.time}</p>
                <p className={""}>{i.content}</p>
            </div>)
        }
        <div></div>
    </div>
}