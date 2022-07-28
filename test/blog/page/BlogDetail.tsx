import Container from "../component/Container";

interface Props {
    title: string,
    html: string
}

export default function BlogDetail(props: Props) {
    return <Container>
        <div>
            <h2> {props.title} </h2>
            <div>
                {props.html}
            </div>
        </div>
    </Container>

}