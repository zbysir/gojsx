export default function Form(props) {
  let x = 1
  return <form b={2} className="form" style={{'fontSize': '1px', padding: '2px'}}>{props.children.map(i => i)} x:{x}</form>
}
