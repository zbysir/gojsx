import Form from "./Form";

export default function App(props) {
  return <div className="bg-red-50 border-black">
    a /2
    <Form className="red block" style={{padding: '1px'}}> f {props.a ? (<>
      <li>a</li>
    </>) : 'b'}</Form>
  </div>
}