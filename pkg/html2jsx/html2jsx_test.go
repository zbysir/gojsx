package html2jsx

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParser(t *testing.T) {
	cases := []struct {
		In   string
		Out  string
		Name string
	}{
		{
			In: `<hr />
        <h2 id="a-1"    bb="r" style="color: red">{a: 1}</h2>
        <Name b="1" name="{{ a: 1 }}" c="2"/>
        <p>{ffff: ffdaf}
        &lt;&gt;
        dfafefdf: fwe :</p>
        <p>f{}fsdfsdfas d{}</p>
        <p>fsd&lt;&gt;&lt;@EOI3u4iuO#$U#($U#94u8u8</p>
        <?fdf>
        
        "'""'""
        ""
        "
        
        ##￥77&￥&￥&7&&&&4uhefuhwf c$&&$
        ;;;
        <><<<><?>
        <p><Toc items = {toc}></Toc></p>
        <p>有不闭合的标签，如 <code>&lt;meta charset=&quot;UTF-8&quot;&gt; </code></p>
        <p>我们要渲染的模板是这个样子的</p>
        <pre><code class="language-vue">&lt;template&gt;
          &lt;div&gt;
            &lt;span class=&quot;bg-gray&quot; :class=&quot;cus_class&quot; :style=&quot;{'font-size': fontSize+'px'}&quot;&gt; {{msg}} &lt;/span&gt;
          &lt;/div&gt;
        &lt;/template&gt;
        </code></pre>
        <h2 id="h2">h2</h2>`,
			Out: `<hr />
        <h2 id="a-1"    bb="r" style="color: red">&#123;a: 1&#125;</h2>
        <name b="1" name="{{ a: 1 }}" c="2"/>
        <p>&#123;ffff: ffdaf&#125;
        &lt;&gt;
        dfafefdf: fwe :</p>
        <p>f&#123;&#125;fsdfsdfas d&#123;&#125;</p>
        <p>fsd&lt;&gt;&lt;@EOI3u4iuO#$U#($U#94u8u8</p>
        &lt;?fdf&gt;
        
        "'""'""
        ""
        "
        
        ##￥77&￥&￥&7&&&&4uhefuhwf c$&&$
        ;;;
        &lt;&gt;&lt;&lt;&lt;&gt;&lt;?&gt;
        <p><toc items ="{toc}"></toc></p>
        <p>有不闭合的标签，如 <code>&lt;meta charset=&quot;UTF-8&quot;&gt; </code></p>
        <p>我们要渲染的模板是这个样子的</p>
        <pre dangerouslySetInnerHTML={{ __html: "<code class=\"language-vue\">&lt;template&gt;\n          &lt;div&gt;\n            &lt;span class=&quot;bg-gray&quot; :class=&quot;cus_class&quot; :style=&quot;{'font-size': fontSize+'px'}&quot;&gt; {{msg}} &lt;/span&gt;\n          &lt;/div&gt;\n        &lt;/template&gt;\n        </code>" }}></pre>
        <h2 id="h2">h2</h2>`,
			Name: "",
		},
		{
			In: `<pre><code class="language-vue">&lt;template&gt;
          &lt;div&gt;
            &lt;span class=&quot;bg-gray&quot; :class=&quot;cus_class&quot; :style=&quot;{'font-size': fontSize+'px'}&quot;&gt; {{msg}} &lt;/span&gt;
          &lt;/div&gt;
        &lt;/template&gt;
        </code></pre>`,
			Out:  `<pre dangerouslySetInnerHTML={{ __html: "<code class=\"language-vue\">&lt;template&gt;\n          &lt;div&gt;\n            &lt;span class=&quot;bg-gray&quot; :class=&quot;cus_class&quot; :style=&quot;{'font-size': fontSize+'px'}&quot;&gt; {{msg}} &lt;/span&gt;\n          &lt;/div&gt;\n        &lt;/template&gt;\n        </code>" }}></pre>`,
			Name: "Code",
		},
		{
			In:   `<Toc items = {toc} a="1" disabled c={aff" ></Toc>`,
			Out:  `<toc items ="{toc}" a="1" disabled c="{aff\"" ></toc>`,
			Name: "Jsx",
		},
		{
			In:   `<pre/> <h1></h1> <pre/>`,
			Out:  `<pre/> <h1></h1> <pre/>`,
			Name: "pre",
		},
		{
			In:   `<></> </>`,
			Out:  `&lt;&gt;&lt;/&gt; &lt;/&gt;`,
			Name: "empty",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			cover := ctx{debug: true}
			var out bytes.Buffer
			err := cover.Covert(bytes.NewBufferString(c.In), &out, false)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, c.Out, out.String())
		})
	}

}
