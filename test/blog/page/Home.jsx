import Container from "../component/Container";

export default function Home(props) {
  return <Container>
    <div>
      <h2> 最新 Blogs </h2>
      <ul className="list-disc list-inside">
        {
          props.blogs.map(i => (
            <li><a href="./detail">{i.name}</a></li>
          ))
        }
      </ul>
    </div>
  </Container>
}