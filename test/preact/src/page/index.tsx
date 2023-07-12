import {useState} from 'preact/hooks';
import Nav from "./nav.tsx";

function List({items}) {
    return <div>
        {
            items.map(item => <div>{item}</div>)
        }
    </div>
}
const itemsd = [1, 2, 3]
for (let i = 0; i < 10000; i++) {
    itemsd.push(i)
}
// itemsd.sort(function() {
//     return (0.5-Math.random());
// });
export default function Root() {
    const [count, setCount] = useState(1)

    const [items, setItems] = useState(itemsd)

    return <div>
        <Nav/>
        <h1 onClick={() => {
            console.log('click');
            items.sort(function() {
                return (0.5-Math.random());
            });
            setItems(items)

            setCount(count + 1)
        }}>Click Me!</h1>
        <h2>{count}</h2>
        <List items={items}></List>
    </div>
}
