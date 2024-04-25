const Frame = ({ node }) => {
  return (
    <div
      data-hydrate={node.hydrate_id ? node.hydrate_id : undefined}
      className={[node.type, node.id, node.classname, node.props._class].join(' ')}
    >
      {node.props.link ? (
        <a
          href={
            node.props.linkType === 'system'
              ? '/' + node.props.link + '/#' + node.props.linkSection
              : node.props.link
          }
          target={node.props.newTarget}
        >
          {node.children.map((ch) => (
            <Comp type={ch.type} node={ch} parentProps={node.props} />
          ))}
        </a>
      ) : (
        <>
          {node.props.fillType === 'image' && node.props.img.src ? (
            <div className="frame-img-box">
              <img
                style={{
                  width: '100%',
                  height: '100%',
                  objectFit: node.props.img.objectFit ? node.props.img.objectFit : 'cover',
                  objectPosition: node.props.img.position ? node.props.img.position : 'center',
                }}
                src={node.props.img.src}
                alt={node.props.img.alt}
              />
            </div>
          ) : null}
          {node.children?.map((ch) => (
            <Comp type={ch.type} node={ch} parentProps={node.props} />
          ))}
        </>
      )}
    </div>
  );
};


const Breakpoint = ({ node }) => {
  return (
    <div className={`breakpoint ${node.id}`} style={{ width: '100%', left: 0, top: 0, position: 'relative' }}>
      {node.children.map((ch) => (
        <Comp type={ch.type} node={ch} parentProps={node.props} />
      ))}
    </div>
  );
};

function Text({ node }) {
  return <div className={`text ${node.id}`} dangerouslySetInnerHTML={{__html:node.props.text}}></div>;
}

function Comp({ type, node, parentProps }) {
  switch (type) {
    case 'frame':
      return <Frame node={node} parentProps={parentProps} />;
    case 'breakpoint':
      return <Breakpoint node={node}></Breakpoint>;
    case 'text':
      return <Text node={node}></Text>;
  }


  return 'uncase type - ' + type;
}

export default function Index({ root, lang, title, style, hydrate_json }) {

  return <html lang={lang}>
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>{title}</title>
    <link
      rel="stylesheet"
      href="https://cdn.jsdelivr.net/gh/iconoir-icons/iconoir@main/css/iconoir.css"
    />

    <style>
      {`   * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }
    .frame-img-box{
    font-size: 0;
  }`}
    </style>

    <style dangerouslySetInnerHTML={{ __html: style }}></style>

  </head>

  <body>
  <Comp type={root.type} node={root}></Comp>;

  <script dangerouslySetInnerHTML={{ __html: 'window.__hydrate = ' + hydrate_json }}></script>

  </body>
  </html>
}
