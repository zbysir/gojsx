export default function Form({children, c}) {
  let x = 1
  x++
  console.log('form', c)
  return <div b={2} className="form" style={{'fontSize': '1px', padding: '2px'}}>
    {children.map(i => i)} x:{x}
    c: {c}
  </div>
}
