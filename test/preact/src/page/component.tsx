// import {Fragment, jsx} from "react/jsx-runtime";

import {ComponentType} from "preact";
import {useEffect, useState} from "preact/hooks";

let Text = (props) => {
    return <div {...props} >
        {props.text}
    </div>
}

let Col = (props) => {
    return <div {...props} style={{...props.style, display: "flex", flexDirection: ""}}>
        {props.children}
    </div>
}

let Row = (props) => {
    return <div {...props} >
        {props.children}
    </div>
}

let Button = (props) => {
    props = {...props}
    switch (props.variant) {
        case "ghost":
            props.style = {
                ...props.style,
                background: "transparent",
            }
            break
        default:
            props.style = {
                ...props.style,
            }
    }
    return <button {...props} >
        {props.children ? props.children : props.text}
    </button>
}

let ComponentX = ({variant = "primary", variants, children}) => {
    const [isHovered, setIsHovered] = useState(false);
    const [count, setCount] = useState(1);
    let main = {...children}
    let hover = variants[variant + "_hover"];
    // main.props = {
    //     ...main.props,
    //     style: {
    //         ...main.props?.style,
    //         // color: "red"
    //     }
    // }
    // useEffect(()=>{
    //     setCount(count+1)
    // },[isHovered])
    if (hover) {
        main.props = {
            ...main.props,
            onMouseEnter: () => {
                console.log('onMouseEnter')
                setIsHovered(true)
            },
            onMouseLeave: () => {
                setIsHovered(false)
            },
            style: {
                ...main.props?.style,
                // color: "red"
            }
        }

        if (isHovered) {
            // 递归修改子组件属性
            console.log('xxxx', main)

            if (hover[main.props?.id]) {
                main = {
                    ...main,
                    props: {
                        ...main.props,
                        ...hover[main.props?.id].props,
                        id: Math.random()
                    },
                }

                // console.log('after', main)
                // setCount(count)
            }
        }
    }


    return main
}

function withComponent(Component: ComponentType<any>, {
    type
}) {
    return function (props) {
        return <Component {...props} data-component-type={type}/>
    }
}

const Components = {
    "row": withComponent(Row, {type: "row"}),
    "col": withComponent(Col, {type: "col"}),
    "text": withComponent(Text, {type: "text"}),
    "button": withComponent(Button, {type: "button"}),
    "component": withComponent(ComponentX, {type: "component"}),
}

// 以后用这个数据来生成 jsx 代码。
//
// nodeTree
const nodeTree = [
    {
        type: "_root",
        props: {
            children: [
                {
                    type: "row",
                    props: {
                        style: {background: "#d7ffbf"}, children: [
                            {type: "text", id: "2334", props: {style: {}, text: "line 1"}}]
                    }
                },
                {
                    type: "col",
                    props: {
                        style: {background: "#ffa09c"},
                        children:
                            [
                                {type: "text", id: "2336", props: {style: {flex: "1 1 0%"}, text: "line 2 left"}},
                                {type: "text", id: "2337", props: {style: {flex: "1 1 0%"}, text: "line 2 right"}}
                            ]
                    }
                },
                {
                    type: "text", props: {style: {background: "#ccdcff", flex: "1 1 0%"}, text: "line 3"},
                },
                {
                    type: "button",
                    props: {style: {background: "#6d8eff", padding: "8px 16px"}, text: "btn", variant: ""}
                },
                {
                    type: "button",
                    props: {style: {background: "#6d8eff", padding: "8px 16px"}, text: "ghost", variant: "ghost"}
                },
                {
                    type: "component",
                    props: {
                        // style: {
                        //     background: "#94ff8f", padding: "8px 16px"
                        // },
                        children: {
                            type: "col",
                            id: "col-1",
                            props: {
                                style: {background: "#ffa09c", transition: "background 1s"},
                                children:
                                    [
                                        {
                                            type: "text",
                                            id: "text-1",
                                            props: {style: {flex: "1 1 0%"}, text: "line 2 left"}
                                        },
                                        {type: "text", props: {style: {flex: "1 1 0%"}, text: "line 2 right"}}
                                    ]
                            }
                        }
                        ,
                        variants: {
                            primary_hover: {
                                "col-1": {
                                    props: {
                                        style: {background: "#84b1ff"}
                                    }
                                }
                            },
                        },
                        variant: "primary"
                    }
                },
            ]
        }
    },
]

// <Tab a=b c=d style={{}} clasName="" />

class NodeTree {
    private nodeIdMap: {};

    constructor(nodeTree) {
        const nodeIdMap = {}
        nodeTree.forEach((node) => {
            nodeIdMap[node.id] = node
        })

        this.nodeIdMap = nodeIdMap
    }

    public nodeToJsx(node) {
        const {type, id, props} = node

        let children = null
        if (props.children) {
            if (Array.isArray(props.children)) {
                children = props.children.map((node) => {
                    return this.nodeToJsx(node)
                })
            } else {
                children = this.nodeToJsx(props.children)
            }
        }
        if (type === "_root") {
            return <>{children}</>
        }

        // if (props.variants) {
        //     Object.keys(props.variants).forEach((key) => {
        //         props.variants[key] = this.nodeToJsx(props.variants[key])
        //     })
        // }

        let C = Components[type];
        return <C id={id} {...props} children={children}/>
    }
}

// interface
function Pages() {
    let nodeTree1 = new NodeTree(nodeTree);
    return nodeTree1.nodeToJsx(nodeTree[0])
}

export default function Component() {
    return <div>
        <Pages/>
    </div>
}

