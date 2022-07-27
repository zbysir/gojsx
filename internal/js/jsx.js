export function jsx(nodeName, attributes) {
  if (typeof nodeName === 'string') {
    return {
      nodeName,
      attributes,
    }
  } else {
    let n = nodeName(attributes)
    let na = n.attributes
    if (na) {
      if (attributes.className) {
        if (na['className']) {
          na['className'] = [na['className'], attributes.className]
        } else {
          na['className'] = attributes.className
        }
      }
      if (attributes.style) {
        na['style'] = Object.assign(na['style'], attributes.style)
      }
    } else {
      na = {}
      if (attributes.className) {
        na['className'] = attributes.className
      }
      if (attributes.style) {
        na['style'] = attributes.style
      }
    }
    n.attributes = na
    return n
  }
}

export function jsxs(nodeName, attributes) {
  // console.log('jsxs', nodeName, JSON.stringify(attributes))
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
