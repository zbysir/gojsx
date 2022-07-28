export default function Header(props) {
  return <div
    className="container mx-auto p-6 bg-white rounded-xl shadow-md flex items-center space-x-4"
  >
    Hello: {props.name}
  </div>
}