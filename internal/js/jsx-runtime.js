export function jsx(nodeName, attributes) {
  if (typeof nodeName === 'string') {
    return {
      nodeName,
      attributes,
    }
  } else {
    return nodeName(attributes)
  }
}

export function jsxs(nodeName, attributes) {
  let x = jsx(nodeName, attributes)
  x.jsxs = true
  return x
}

export function Fragment(args) {
  return {
    nodeName: "",
    attributes: args
  }
}
