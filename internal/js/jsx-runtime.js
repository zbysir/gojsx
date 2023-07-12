export function jsx(nodeName, attributes) {
    if (typeof nodeName === 'string') {
        return {
            nodeName, attributes,
        }
    } else {
        return nodeName(attributes)
    }
}

export function jsxs(nodeName, attributes) {
    return jsx(nodeName, attributes)
}

export function Fragment(args) {
    return {
        nodeName: "", attributes: args
    }
}
