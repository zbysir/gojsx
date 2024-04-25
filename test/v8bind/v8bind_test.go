package v8bind

import (
	"encoding/json"
	"fmt"
	"github.com/zbysir/gojsx"
	"os"
	v8 "rogchap.com/v8go"
	"testing"
	"time"
)

func TestV8(t *testing.T) {
	ctx := v8.NewContext()
	bs, err := os.ReadFile("./test.js")
	if err != nil {
		t.Fatal(err)
		return
	}

	_, err = ctx.RunScript(`

function jsx(nodeName, attributes) {
    if (typeof nodeName === 'string') {
        return {
            nodeName, attributes,
        }
    } else {
        return nodeName(attributes)
    }
}

function jsxs(nodeName, attributes) {
    return jsx(nodeName, attributes)
}

function Fragment(args) {
    return {
        nodeName: "", attributes: args
    }
}


function require(path){
return {
 jsx: jsx,

jsxs: jsxs,

Fragment: Fragment
}
}
`, "runtime")
	if err != nil {
		t.Fatal(err)
	}

	//v8.NewObjectTemplate().NewInstance()

	obj, err := v8.NewObjectTemplate(ctx.Isolate()).NewInstance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tr := gojsx.NewEsBuildTransform(gojsx.EsBuildTransformOptions{})
	err = ctx.Global().Set("module", obj)
	if err != nil {
		t.Fatal(err)
	}
	bs, err = tr.Transform("x.tsx", bs, gojsx.TransformerFormatCommonJS)
	if err != nil {
		t.Fatal(err)
	}

	res, err := ctx.RunScript(fmt.Sprintf("%s; Index", string(bs)), "1")
	if err != nil {
		t.Fatal("Unexpected error on eval,", err)
	}
	f, _ := res.AsFunction()

	start := time.Now()

	//module, err := v8.NewValue(ctx.Isolate(), map[string]any{})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//fun := v8.NewFunctionTemplate(ctx.Isolate(), func(info *v8.FunctionCallbackInfo) *v8.Value {
	//	obj, _ := v8.NewObjectTemplate(ctx.Isolate()).NewInstance(ctx)
	//	return obj.Value
	//}).GetFunction(ctx)

	//err = ctx.Global().Set("require", fun)
	//if err != nil {
	//	t.Fatal(err)
	//}

	//t.Logf("bs: %s", bs)
	//ctx.RunScript()

	data, err := v8.JSONParse(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("data:%+v", data)

	v, err := f.Call(data, data)
	if err != nil {
		t.Fatal(err)
	}
	d := time.Since(start)
	t.Logf("d:%+v", d)
	t.Logf("%+v", res)
	json, err := v.MarshalJSON()
	t.Logf("v: %s", json)
	//
	//exports, err := obj.Get("exports")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//t.Logf("exports: %+v", exports.Object())

}

func BenchmarkName(b *testing.B) {
	ctx := v8.NewContext()
	bs, err := os.ReadFile("./test.js")
	if err != nil {
		b.Fatal(err)
		return
	}

	_, err = ctx.RunScript(`

function jsx(nodeName, attributes) {
    if (typeof nodeName === 'string') {
        return {
            nodeName, attributes,
        }
    } else {
        return nodeName(attributes)
    }
}

function jsxs(nodeName, attributes) {
    return jsx(nodeName, attributes)
}

function Fragment(args) {
    return {
        nodeName: "", attributes: args
    }
}


function require(path){
return {
 jsx: jsx,

jsxs: jsxs,

Fragment: Fragment
}
}
`, "runtime")
	if err != nil {
		b.Fatal(err)
	}

	//v8.NewObjectTemplate().NewInstance()

	obj, err := v8.NewObjectTemplate(ctx.Isolate()).NewInstance(ctx)
	if err != nil {
		b.Fatal(err)
	}
	tr := gojsx.NewEsBuildTransform(gojsx.EsBuildTransformOptions{})
	err = ctx.Global().Set("module", obj)
	if err != nil {
		b.Fatal(err)
	}
	bs, err = tr.Transform("x.tsx", bs, gojsx.TransformerFormatCommonJS)
	if err != nil {
		b.Fatal(err)
	}

	res, err := ctx.RunScript(fmt.Sprintf("%s; Index", string(bs)), "1")
	if err != nil {
		b.Fatal("Unexpected error on eval,", err)
	}
	f, _ := res.AsFunction()
	data, err := v8.JSONParse(ctx, data)
	if err != nil {
		b.Fatal(err)
	}
	//data.Object().Set()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = f.Call(data, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 1,202,041 ns/op
func BenchmarkJSON(b *testing.B) {
	ctx := v8.NewContext()
	for i := 0; i < b.N; i++ {
		v8.JSONParse(ctx, data)
	}
}

// 3,229 ns/op
func BenchmarkJSONSample(b *testing.B) {
	ctx := v8.NewContext()
	for i := 0; i < b.N; i++ {
		v8.JSONParse(ctx, sampleData)
	}
}

// 18,634 ns/op
// 反而更慢了，猜测是需要多次调用 c 的原因。
func BenchmarkToValueSample(b *testing.B) {
	v := map[string]interface{}{}
	json.Unmarshal([]byte(sampleData), &v)
	ctx := v8.NewContext()
	ot := v8.NewObjectTemplate(ctx.Isolate())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toValue(ctx.Isolate(), ot, ctx, v)
	}
}

// 1,611,960 ns/op
func BenchmarkGOJSON(b *testing.B) {
	//ctx := v8.NewContext()
	for i := 0; i < b.N; i++ {
		json.Unmarshal([]byte(data), &map[string]interface{}{})
	}
}

func toValue(iso *v8.Isolate, pt *v8.ObjectTemplate, ctx *v8.Context, v any) *v8.Value {
	switch v.(type) {
	case string,
		int32,
		uint32,
		int64,
		uint64,
		bool,
		float64:
		x, err := v8.NewValue(iso, v)
		if err != nil {
			panic(err)
		}
		return x
	case map[string]any:
		obj, _ := pt.NewInstance(ctx)
		for k, v := range v.(map[string]any) {
			obj.Set(k, toValue(iso, pt, ctx, v))
		}

		return obj.Value
	default:
		panic(fmt.Sprintf("unsupported type: %T", v))
	}
}

const sampleData = `{
    "root": {
        "id": "lZRRDONbpAi",
        "vid": "lZRRDONbpAi",
        "type": "breakpoint"
    },
    "abc": {
        "id": 1,
        "age": 2
    }
}
`

const data = `{
  "root": {
    "id": "lZRRDONbpAi",
    "vid": "lZRRDONbpAi",
    "hydrate_id": "",
    "type": "breakpoint",
    "children_ids": [
      "mhlFKZIsbSa"
    ],
    "props": {
      "align": "center",
      "background": "#E9ECF1",
      "bgColor": "rgba(255,255,255,1)",
      "direction": "column",
      "display": "flex",
      "gap": 0,
      "height": "auto",
      "left": -3034,
      "pin": {
        "top": 1,
        "left": 1
      },
      "position": "",
      "top": -3319,
      "width": "1600px",
      "wrap": "nowrap"
    },
    "children": [
      {
        "id": "mhlFKZIsbSa",
        "vid": "mhlFKZIsbSa",
        "hydrate_id": "",
        "type": "frame",
        "children_ids": [
          "mfJAradyRRC",
          "mfJtxplzrle",
          "mgkYQniMxVN",
          "mfJumyOUAte",
          "mgykUHPtsfU",
          "mgymoGzJkCc"
        ],
        "props": {
          "align": "center",
          "direction": "column",
          "display": "flex",
          "gap": 0,
          "height": "auto",
          "left": 0,
          "padding": {
            "value": 0,
            "perSide": true
          },
          "top": 0,
          "width": "auto",
          "wrap": "nowrap"
        },
        "children": [
          {
            "id": "mfJAradyRRC",
            "vid": "mfJAradyRRC",
            "hydrate_id": "",
            "type": "frame",
            "children_ids": [
              "mgjuNkALlLn"
            ],
            "props": {
              "background": "none 50% 0% / cover no-repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
              "borderBottom": "0px none rgb(255, 255, 255)",
              "borderLeft": "0px none rgb(255, 255, 255)",
              "borderRadius": "0px",
              "borderRight": "0px none rgb(255, 255, 255)",
              "borderTop": "0px none rgb(255, 255, 255)",
              "direction": "column",
              "display": "flex",
              "gap": 0,
              "height": "auto",
              "left": 523.9842914628912,
              "padding": {
                "top": 24,
                "left": 0,
                "right": 0,
                "bottom": 24,
                "perSide": true
              },
              "position": "",
              "top": 0,
              "width": "auto",
              "wrap": "nowrap"
            },
            "children": [
              {
                "id": "mgjuNkALlLn",
                "vid": "mgjuNkALlLn",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mgyGixoPfWW"
                ],
                "props": {
                  "align": "center",
                  "background": "none 0% 0% / auto repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                  "borderBottom": "0px none rgb(25, 25, 25)",
                  "borderLeft": "0px none rgb(25, 25, 25)",
                  "borderRadius": "0px",
                  "borderRight": "0px none rgb(25, 25, 25)",
                  "borderTop": "0px none rgb(25, 25, 25)",
                  "direction": "column",
                  "display": "flex",
                  "gap": 10,
                  "height": "auto",
                  "left": 0,
                  "padding": {
                    "top": 15,
                    "left": 0,
                    "right": 0,
                    "bottom": 15,
                    "perSide": true
                  },
                  "pin": {
                    "top": 1,
                    "left": 1
                  },
                  "position": "",
                  "text": "\u003cp style=\"font-size: 36px;color:rgb(255, 255, 255);line-height:47.9988px;text-align:center;font-family:li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;\" class=\"weave-text \"\u003e理想增程电动平台\u003c/p\u003e",
                  "top": 0,
                  "width": "auto",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mgyGixoPfWW",
                    "vid": "mgyGixoPfWW",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgkXFDTNbLS",
                      "mgjuNkALlLm"
                    ],
                    "props": {
                      "direction": "column",
                      "display": "flex",
                      "gap": 10,
                      "height": "auto",
                      "left": 524.0001146643372,
                      "padding": {
                        "value": 0,
                        "perSide": false
                      },
                      "top": 15.000007525702358,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgkXFDTNbLS",
                        "vid": "mgkXFDTNbLS",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "text": "\u003cp style=\"opacity: 1;color: rgb(25, 25, 25);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 46px;font-weight: 400;line-height: 71.9992px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(25, 25, 25);border-right: 0px none rgb(25, 25, 25);border-bottom: 0px none rgb(25, 25, 25);border-left: 0px none rgb(25, 25, 25);border-radius: 0px;align-items: normal\"\u003e提供更便捷的能源解决方案\u003c/p\u003e",
                          "top": 0
                        }
                      },
                      {
                        "id": "mgjuNkALlLm",
                        "vid": "mgjuNkALlLm",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "text": "\u003cp style=\"opacity: 1;color: rgb(102, 102, 102);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(102, 102, 102);border-right: 0px none rgb(102, 102, 102);border-bottom: 0px none rgb(102, 102, 102);border-left: 0px none rgb(102, 102, 102);border-radius: 0px;align-items: normal\"\u003e增程和纯电并行，通过可再生能源革命，大规模替代燃油车。\u003c/p\u003e",
                          "top": 0
                        }
                      }
                    ]
                  }
                ]
              }
            ]
          },
          {
            "id": "mfJtxplzrle",
            "vid": "mfJtxplzrle",
            "hydrate_id": "",
            "type": "frame",
            "children_ids": [
              "mfJtxplzrld",
              "mfJtxplzrkV"
            ],
            "props": {
              "background": "none 0% 0% / auto repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
              "borderBottom": "0px none rgb(255, 255, 255)",
              "borderLeft": "0px none rgb(255, 255, 255)",
              "borderRadius": "0px",
              "borderRight": "0px none rgb(255, 255, 255)",
              "borderTop": "0px none rgb(255, 255, 255)",
              "direction": "row",
              "display": "flex",
              "gap": 20.000056216591133,
              "height": "1fr",
              "left": 99.99973980244704,
              "maxWidth": "1400px",
              "pin": {
                "top": 1,
                "left": 1
              },
              "position": "",
              "top": 187.98438684583607,
              "width": "1fr",
              "wrap": "nowrap"
            },
            "children": [
              {
                "id": "mfJtxplzrld",
                "vid": "mfJtxplzrld",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mfJtxplzrlc",
                  "mfQeWCaAMEi"
                ],
                "props": {
                  "background": "url(\"https://p.ampmake.com/lilibrary/858858135568871/573ff2cc-5c53-4113-8e24-92aacf1ce49d.jpg@d_progressive\") 50% 50% / cover no-repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "4px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "distribution": "space-between",
                  "fillType": "image",
                  "gap": 10,
                  "height": "761px",
                  "img": {
                    "src": "user/0/lPNhRrFCGEZ__m-理想增程电动平台.jpg",
                    "objectFit": "cover"
                  },
                  "left": 160,
                  "position": "",
                  "top": 0,
                  "width": 1,
                  "width_unit": "fr",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mfJtxplzrlc",
                    "vid": "mfJtxplzrlc",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgZooedCAKq"
                    ],
                    "props": {
                      "align": "center",
                      "background": "none 0% 0% / auto repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "row",
                      "display": "flex",
                      "gap": 0,
                      "height": "auto",
                      "left": 0,
                      "padding": {
                        "top": 80,
                        "left": 15,
                        "right": 15,
                        "bottom": 15,
                        "perSide": true
                      },
                      "position": "",
                      "top": 0,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgZooedCAKq",
                        "vid": "mgZooedCAKq",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mfJtxplzrlb",
                          "mfJtxplzrla",
                          "mfJtxplzrkY"
                        ],
                        "props": {
                          "align": "center",
                          "direction": "column",
                          "display": "flex",
                          "gap": 20,
                          "height": "auto",
                          "left": 136.99993760508596,
                          "padding": {
                            "value": 0,
                            "perSide": false
                          },
                          "position": "",
                          "top": 80.00001163063081,
                          "width": "auto",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mfJtxplzrlb",
                            "vid": "mfJtxplzrlb",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "height": "auto",
                              "left": 201,
                              "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 36px;font-weight: 400;line-height: 47.9988px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-radius: 0px;align-items: normal\"\u003e理想增程电动平台\u003c/p\u003e",
                              "top": 80,
                              "width": "auto"
                            }
                          },
                          {
                            "id": "mfJtxplzrla",
                            "vid": "mfJtxplzrla",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "height": "auto",
                              "left": 137,
                              "text": "\u003cp style=\"opacity: 1;color: rgba(255, 255, 255, 0.7);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgba(255, 255, 255, 0.7);border-right: 0px none rgba(255, 255, 255, 0.7);border-bottom: 0px none rgba(255, 255, 255, 0.7);border-left: 0px none rgba(255, 255, 255, 0.7);border-radius: 0px;align-items: normal\"\u003e城市用电，长途发电，露营供电，让电动车没有里程焦虑。\u003c/p\u003e",
                              "top": 144,
                              "width": "auto"
                            }
                          },
                          {
                            "id": "mfJtxplzrkY",
                            "vid": "mfJtxplzrkY",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "height": "auto",
                              "left": 317,
                              "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 21px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-radius: 0px;align-items: center\"\u003e了解更多\u003c/p\u003e",
                              "top": 192.00007916093324,
                              "width": "auto"
                            }
                          }
                        ]
                      }
                    ]
                  },
                  {
                    "id": "mfQeWCaAMEi",
                    "vid": "mfQeWCaAMEi",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mfJtxplzrkW"
                    ],
                    "props": {
                      "direction": "column",
                      "display": "flex",
                      "gap": 0,
                      "height": "auto",
                      "left": 248.24230380630888,
                      "padding": {
                        "value": 15,
                        "perSide": false
                      },
                      "top": 733.0000285292533,
                      "width": "1fr",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mfJtxplzrkW",
                        "vid": "mfJtxplzrkW",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgba(255, 255, 255, 0.5);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgba(255, 255, 255, 0.5);border-right: 0px none rgba(255, 255, 255, 0.5);border-bottom: 0px none rgba(255, 255, 255, 0.5);border-left: 0px none rgba(255, 255, 255, 0.5);border-radius: 0px;align-items: normal\"\u003e*理想L系列全系车型标配。\u003c/p\u003e",
                          "top": 0
                        }
                      }
                    ]
                  }
                ]
              },
              {
                "id": "mfJtxplzrkV",
                "vid": "mfJtxplzrkV",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mfJtxplzrkU",
                  "mfQeZJeOkvC"
                ],
                "props": {
                  "background": "url(\"https://p.ampmake.com/lilibrary/85886072267728/c9b239c6-ad49-4f1d-ace7-a087b02b0235.jpg@d_progressive\") 50% 50% / cover no-repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "4px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "distribution": "space-between",
                  "fillType": "image",
                  "gap": 10,
                  "height": "761px",
                  "img": {
                    "src": "user/0/lPNhTBmTiEZ__PC_纯电电能.jpg"
                  },
                  "left": 933,
                  "position": "",
                  "top": 0,
                  "width": 1,
                  "width_unit": "fr",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mfJtxplzrkU",
                    "vid": "mfJtxplzrkU",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgZouKFNQQy"
                    ],
                    "props": {
                      "align": "center",
                      "background": "none 0% 0% / auto repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "row",
                      "display": "flex",
                      "fillType": "color",
                      "gap": 0,
                      "height": "auto",
                      "left": 0,
                      "padding": {
                        "top": 80,
                        "left": 15,
                        "right": 15,
                        "bottom": 60,
                        "perSide": true
                      },
                      "position": "",
                      "top": 0,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgZouKFNQQy",
                        "vid": "mgZouKFNQQy",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgyGSXYiphK",
                          "mfJtxplzrkQ"
                        ],
                        "props": {
                          "align": "center",
                          "direction": "column",
                          "display": "flex",
                          "gap": 26,
                          "height": "auto",
                          "left": 150.99209965452906,
                          "padding": {
                            "value": 0,
                            "perSide": false
                          },
                          "position": "",
                          "top": 80.00001163063081,
                          "width": "auto",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgyGSXYiphK",
                            "vid": "mgyGSXYiphK",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mfJtxplzrkT",
                              "mfJtxplzrkS"
                            ],
                            "props": {
                              "direction": "column",
                              "display": "flex",
                              "gap": 10,
                              "height": "auto",
                              "left": 0,
                              "padding": {
                                "value": 0,
                                "perSide": false
                              },
                              "position": "",
                              "top": 0,
                              "width": "auto",
                              "wrap": "nowrap"
                            },
                            "children": [
                              {
                                "id": "mfJtxplzrkT",
                                "vid": "mfJtxplzrkT",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 151,
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 36px;font-weight: 400;line-height: 47.9988px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-radius: 0px;align-items: normal\"\u003e理想800伏高压纯电平台\u003c/p\u003e",
                                  "top": 80
                                }
                              },
                              {
                                "id": "mfJtxplzrkS",
                                "vid": "mfJtxplzrkS__mapxGgRqUpu",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "hide-lZRRDONbpAi hide-maxgJZzNUPm",
                                  "left": 224,
                                  "pin": {
                                    "top": 1,
                                    "left": 1,
                                    "right": 0,
                                    "bottom": 0
                                  },
                                  "text": "\u003cp style=\"font-size: 16px;color:rgba(255, 255, 255, 0.7);line-height:28px;text-align:center;font-family:li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;\" class=\"weave-text \"\u003e5C电池充电12分钟续航500公里， 将充电速度带入“5G时代”。\u003c/p\u003e",
                                  "top": 138
                                }
                              },
                              {
                                "id": "mfJtxplzrkS",
                                "vid": "mfJtxplzrkS",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "hide-mapxGgRqUpu",
                                  "left": 224,
                                  "pin": {
                                    "top": 1,
                                    "left": 1,
                                    "right": 0,
                                    "bottom": 0
                                  },
                                  "text": "\u003cp style=\"opacity: 1;color: rgba(255, 255, 255, 0.7);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgba(255, 255, 255, 0.7);border-right: 0px none rgba(255, 255, 255, 0.7);border-bottom: 0px none rgba(255, 255, 255, 0.7);border-left: 0px none rgba(255, 255, 255, 0.7);border-radius: 0px;align-items: normal\"\u003e5C电池充电12分钟续航500公里，\u003cbr\u003e将充电速度带入“5G时代”。\u003c/p\u003e",
                                  "top": 138
                                }
                              }
                            ]
                          },
                          {
                            "id": "mfJtxplzrkQ",
                            "vid": "mfJtxplzrkQ",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "left": 166.00775722029562,
                              "position": "",
                              "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 21px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-radius: 0px;align-items: center\"\u003e敬请期待\u003c/p\u003e",
                              "top": 139.99999897376784
                            }
                          }
                        ]
                      }
                    ]
                  },
                  {
                    "id": "mfQeZJeOkvC",
                    "vid": "mfQeZJeOkvC",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mfJtxplzrkO"
                    ],
                    "props": {
                      "direction": "column",
                      "display": "flex",
                      "gap": 0,
                      "height": "auto",
                      "left": 34.50004296491852,
                      "padding": {
                        "value": 15,
                        "perSide": false
                      },
                      "top": 704.9999688709586,
                      "width": "1fr",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mfJtxplzrkO",
                        "vid": "mfJtxplzrkO",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "pin": {
                            "top": 1,
                            "left": 0,
                            "right": 0
                          },
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgba(255, 255, 255, 0.5);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgba(255, 255, 255, 0.5);border-right: 0px none rgba(255, 255, 255, 0.5);border-bottom: 0px none rgba(255, 255, 255, 0.5);border-left: 0px none rgba(255, 255, 255, 0.5);border-radius: 0px;align-items: normal\"\u003e*数据基于最优充电性能自测所得，充电速度和续航里程受车型、车辆状态和使用情况等因素影响有所差异。\u003c/p\u003e",
                          "top": 0,
                          "width": "auto"
                        }
                      }
                    ]
                  }
                ]
              }
            ]
          },
          {
            "id": "mgkYQniMxVN",
            "vid": "mgkYQniMxVN",
            "hydrate_id": "",
            "type": "frame",
            "children_ids": [
              "mgkYWFocFui",
              "mgkZaBDydTS"
            ],
            "props": {
              "direction": "column",
              "display": "flex",
              "gap": 0,
              "height": "auto",
              "left": 0,
              "padding": {
                "top": 0,
                "left": 0,
                "right": 0,
                "bottom": 0,
                "perSide": true
              },
              "position": "",
              "top": 948.9765950931672,
              "width": 1,
              "width_unit": "fr",
              "widthmapxGgRqUpu": "1fr",
              "wrap": "nowrap"
            },
            "children": [
              {
                "id": "mgkYWFocFui",
                "vid": "mgkYWFocFui",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mgyHpplWHci"
                ],
                "props": {
                  "background": "none 50% 0% / cover no-repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "0px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "gap": 0,
                  "height": "auto",
                  "left": 0,
                  "padding": {
                    "top": 72,
                    "left": 0,
                    "right": 0,
                    "bottom": 72,
                    "perSide": true
                  },
                  "top": 0,
                  "width": "auto",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mgyHpplWHci",
                    "vid": "mgyHpplWHci",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgkYWFocFuf",
                      "mgkYWFocFub"
                    ],
                    "props": {
                      "direction": "column",
                      "display": "flex",
                      "gap": 0,
                      "height": "auto",
                      "left": 0,
                      "top": 72.00001901950225,
                      "width": "1fr",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgkYWFocFuf",
                        "vid": "mgkYWFocFuf",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgkYWFocFud"
                        ],
                        "props": {
                          "background": "none 0% 0% / auto repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                          "borderBottom": "0px none rgb(25, 25, 25)",
                          "borderLeft": "0px none rgb(25, 25, 25)",
                          "borderRadius": "0px",
                          "borderRight": "0px none rgb(25, 25, 25)",
                          "borderTop": "0px none rgb(25, 25, 25)",
                          "direction": "column",
                          "display": "flex",
                          "gap": 0,
                          "height": "auto",
                          "left": 224.00013518898004,
                          "padding": {
                            "top": 0,
                            "left": 27,
                            "right": 27,
                            "bottom": 0,
                            "perSide": true
                          },
                          "top": 72.00001901950225,
                          "width": "auto",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgkYWFocFud",
                            "vid": "mgkYWFocFud",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "left": 0,
                              "position": "",
                              "text": "\u003ch1 style=\"font-size: 46px;color:rgba(74,74,74,1);line-height:1.4em;letter-spacing:0.04em;text-decoration:none;\" class=\"weave-text \"\u003e用户和我们一起谈谈理想\u003c/h1\u003e",
                              "top": 0
                            }
                          }
                        ]
                      },
                      {
                        "id": "mgkYWFocFub",
                        "vid": "mgkYWFocFub",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 520.0002038781176,
                          "text": "\u003cp style=\"opacity: 1;color: rgb(102, 102, 102);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(102, 102, 102);border-right: 0px none rgb(102, 102, 102);border-bottom: 0px none rgb(102, 102, 102);border-left: 0px none rgb(102, 102, 102);border-radius: 0px;align-items: normal\"\u003e理想汽车是用户的书房、卧室、客厅，更是孩子们的魔术城堡。\u003c/p\u003e",
                          "top": 143.9999952793322
                        }
                      }
                    ]
                  }
                ]
              },
              {
                "id": "mgkZaBDydTS",
                "vid": "mgkZaBDydTS",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mgrROJEgcgX",
                  "mgrRPCStvmf",
                  "mgrRSSGILHc"
                ],
                "props": {
                  "direction": "row",
                  "display": "flex",
                  "gap": 12,
                  "height": "auto",
                  "left": -867.5000992378532,
                  "top": 170.59374627858045,
                  "width": 1,
                  "width_unit": "fr",
                  "wrap": "wrap"
                },
                "children": [
                  {
                    "id": "mgrROJEgcgX",
                    "vid": "mgrROJEgcgX",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgyHtxjgrNe",
                      "mgrROJEgcgS",
                      "mgZvdBosioO"
                    ],
                    "props": {
                      "align": "flex-start",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "column",
                      "display": "flex",
                      "gap": 19,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "top": 0,
                        "left": 0,
                        "right": 0,
                        "bottom": 0,
                        "perSide": true
                      },
                      "top": 0,
                      "width": 1,
                      "width_unit": "fr",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgyHtxjgrNe",
                        "vid": "mgyHtxjgrNe",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgrROJEgcgW",
                          "mgrROJEgcgV"
                        ],
                        "props": {
                          "direction": "row",
                          "display": "flex",
                          "gap": 20,
                          "height": 1,
                          "height_unit": "fr",
                          "left": 0,
                          "position": "",
                          "top": 0,
                          "width": 1,
                          "width_unit": "fr",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgrROJEgcgW",
                            "vid": "mgrROJEgcgW",
                            "hydrate_id": "",
                            "type": "frame",
                            "props": {
                              "bgColor": "rgb(230, 241, 240)",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "4px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "image",
                              "height": "237px",
                              "img": {
                                "alt": "",
                                "src": "26fae5310bfa7ecbfd8f6a72de503c7b/177e7a6e-bec6-4a8a-bee9-48f76fb4fda6.jpg@d_progressive",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 0,
                              "top": 0,
                              "width": "1fr"
                            }
                          },
                          {
                            "id": "mgrROJEgcgV",
                            "vid": "mgrROJEgcgV",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgrROJEgcgU",
                              "mgrROJEgcgT"
                            ],
                            "props": {
                              "align": "center",
                              "bgColor": "rgb(230, 241, 240)",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "4px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "bottom": 1,
                              "direction": "column",
                              "display": "flex",
                              "fillType": "color",
                              "gap": 52,
                              "height": "1fr",
                              "img": {
                                "alt": "",
                                "src": "",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 247.00015616816518,
                              "padding": {
                                "top": 40,
                                "left": 22,
                                "right": 0,
                                "bottom": 40,
                                "perSide": true
                              },
                              "position": "relative",
                              "top": -1,
                              "width": "1fr",
                              "wrap": "nowrap"
                            },
                            "children": [
                              {
                                "id": "mgrROJEgcgU",
                                "vid": "mgrROJEgcgU",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 22,
                                  "position": "",
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(102, 102, 102);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(102, 102, 102);border-right: 0px none rgb(102, 102, 102);border-bottom: 0px none rgb(102, 102, 102);border-left: 0px none rgb(102, 102, 102);border-radius: 0px;align-items: normal\"\u003e“理想作为我们的移动城堡，也给两个小可爱找到适合他们的空间。”\u003c/p\u003e",
                                  "top": 40
                                }
                              },
                              {
                                "id": "mgrROJEgcgT",
                                "vid": "mgrROJEgcgT",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 32,
                                  "position": "",
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 21px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(153, 153, 153);border-right: 0px none rgb(153, 153, 153);border-bottom: 0px none rgb(153, 153, 153);border-left: 0px none rgb(153, 153, 153);border-radius: 0px;align-items: normal\"\u003e理想汽车用户@dolocy\u003c/p\u003e",
                                  "top": 176
                                }
                              }
                            ]
                          }
                        ]
                      },
                      {
                        "id": "mgrROJEgcgS",
                        "vid": "mgrROJEgcgS",
                        "hydrate_id": "",
                        "type": "frame",
                        "props": {
                          "bgColor": "rgb(230, 241, 240)",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "4px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "fillType": "image",
                          "height": "494px",
                          "img": {
                            "alt": "",
                            "src": "c63bef8d033302feacae3fdf88556228/7e3c43a0-8f4c-4f32-b738-815a287070b5.jpg@d_progressive",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "position": "",
                          "top": 256,
                          "width": 1,
                          "width_unit": "fr"
                        }
                      },
                      {
                        "id": "mgZvdBosioO",
                        "vid": "mgZvdBosioO",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgrROJEgcgR",
                          "mgrROJEgcgO"
                        ],
                        "props": {
                          "align": "center",
                          "direction": "row",
                          "display": "flex",
                          "gap": 20,
                          "height": 236.99995172502577,
                          "left": 0,
                          "position": "",
                          "top": 768.9999895655653,
                          "width": "1fr",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgrROJEgcgR",
                            "vid": "mgrROJEgcgR",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgrROJEgcgQ",
                              "mgrROJEgcgP"
                            ],
                            "props": {
                              "bgColor": "rgb(230, 241, 240)",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "4px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "color",
                              "height": "237px",
                              "img": {
                                "alt": "",
                                "src": "",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 0,
                              "top": 769,
                              "width": "1fr"
                            },
                            "children": [
                              {
                                "id": "mgrROJEgcgQ",
                                "vid": "mgrROJEgcgQ",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 32,
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(102, 102, 102);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(102, 102, 102);border-right: 0px none rgb(102, 102, 102);border-bottom: 0px none rgb(102, 102, 102);border-left: 0px none rgb(102, 102, 102);border-radius: 0px;align-items: normal\"\u003e“带着“理想”，\u003cbr\u003e奔赴理想。”\u003c/p\u003e",
                                  "top": 68
                                }
                              },
                              {
                                "id": "mgrROJEgcgP",
                                "vid": "mgrROJEgcgP",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 32,
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 21px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(153, 153, 153);border-right: 0px none rgb(153, 153, 153);border-bottom: 0px none rgb(153, 153, 153);border-left: 0px none rgb(153, 153, 153);border-radius: 0px;align-items: normal\"\u003e理想汽车用户@旅行的独白\u003c/p\u003e",
                                  "top": 148
                                }
                              }
                            ]
                          },
                          {
                            "id": "mgrROJEgcgO",
                            "vid": "mgrROJEgcgO",
                            "hydrate_id": "",
                            "type": "frame",
                            "props": {
                              "bgColor": "rgb(230, 241, 240)",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "4px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "image",
                              "height": "237px",
                              "img": {
                                "alt": "",
                                "src": "30d725504175293d2098d7abb6a9433a/bd50fac2-d69b-4aa4-a9e1-7fe9192b6907.jpg@d_progressive",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 258,
                              "top": 769,
                              "width": "1fr"
                            }
                          }
                        ]
                      }
                    ]
                  },
                  {
                    "id": "mgrRPCStvmf",
                    "vid": "mgrRPCStvmf",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgrRPCStvme",
                      "mgrRPCStvmd",
                      "mgrRPCStvma"
                    ],
                    "props": {
                      "align": "flex-start",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "column",
                      "display": "flex",
                      "gap": 20,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "top": 0,
                        "left": 0,
                        "right": 0,
                        "bottom": 0,
                        "perSide": true
                      },
                      "top": 0,
                      "width": 1,
                      "width_unit": "fr",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgrRPCStvme",
                        "vid": "mgrRPCStvme",
                        "hydrate_id": "",
                        "type": "frame",
                        "props": {
                          "align": "center",
                          "bgColor": "rgb(230, 241, 240)",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "4px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "direction": "row",
                          "display": "flex",
                          "fillType": "image",
                          "gap": 8,
                          "height": "365px",
                          "img": {
                            "alt": "",
                            "src": "a92edd188586de8d51d24285ea922f29/a7b42ea2-d389-4c61-8d93-91a6f2619cc5.jpg@d_progressive",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "position": "",
                          "top": 0,
                          "width": "1fr",
                          "wrap": "nowrap"
                        }
                      },
                      {
                        "id": "mgrRPCStvmd",
                        "vid": "mgrRPCStvmd",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgrRPCStvmc",
                          "mgrRPCStvmb"
                        ],
                        "props": {
                          "align": "flex-start",
                          "bgColor": "rgb(230, 241, 240)",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "4px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "direction": "column",
                          "display": "flex",
                          "fillType": "color",
                          "gap": 24,
                          "height": "auto",
                          "img": {
                            "alt": "",
                            "src": "",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "padding": {
                            "top": 67,
                            "left": 32,
                            "right": 0,
                            "bottom": 69,
                            "perSide": true
                          },
                          "position": "",
                          "top": 385,
                          "width": "1fr",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgrRPCStvmc",
                            "vid": "mgrRPCStvmc",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "left": 32,
                              "position": "",
                              "text": "\u003cp style=\"opacity: 1;color: rgb(102, 102, 102);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(102, 102, 102);border-right: 0px none rgb(102, 102, 102);border-bottom: 0px none rgb(102, 102, 102);border-left: 0px none rgb(102, 102, 102);border-radius: 0px;align-items: normal\"\u003e“以后的道路，道阻且长；\u003cbr\u003e眼前的生活，诗和远方。”\u003c/p\u003e",
                              "top": 67
                            }
                          },
                          {
                            "id": "mgrRPCStvmb",
                            "vid": "mgrRPCStvmb",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "left": 32,
                              "position": "",
                              "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 21px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(153, 153, 153);border-right: 0px none rgb(153, 153, 153);border-bottom: 0px none rgb(153, 153, 153);border-left: 0px none rgb(153, 153, 153);border-radius: 0px;align-items: normal\"\u003e理想汽车用户@包小秋\u003c/p\u003e",
                              "top": 147
                            }
                          }
                        ]
                      },
                      {
                        "id": "mgrRPCStvma",
                        "vid": "mgrRPCStvma",
                        "hydrate_id": "",
                        "type": "frame",
                        "props": {
                          "bgColor": "rgb(230, 241, 240)",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "4px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "fillType": "image",
                          "height": "365px",
                          "img": {
                            "alt": "",
                            "src": "e912f26b29db89f9516a608a3f34b03b/0a27b6f5-1b62-4d3a-8f74-c159c78f45b5.jpg@d_progressive",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "position": "",
                          "top": 641,
                          "width": "1fr"
                        }
                      }
                    ]
                  },
                  {
                    "id": "mgrRSSGILHc",
                    "vid": "mgrRSSGILHc",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgrRSSGILHb",
                      "mgZvgjUnkaG",
                      "mgrRSSGILGW"
                    ],
                    "props": {
                      "align": "flex-start",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "column",
                      "display": "flex",
                      "gap": 19,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "value": 0,
                        "perSide": false
                      },
                      "top": 0,
                      "width": 1,
                      "width_unit": "fr",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgrRSSGILHb",
                        "vid": "mgrRSSGILHb",
                        "hydrate_id": "",
                        "type": "frame",
                        "props": {
                          "bgColor": "rgb(230, 241, 240)",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "4px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "fillType": "image",
                          "height": "494px",
                          "img": {
                            "alt": "",
                            "src": "e63cc31423114b201f9d9196eb3c010d/814397a5-c1ae-40d2-8e71-b6ea76bd3fca.jpg@d_progressive",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "position": "",
                          "top": 0,
                          "width": "1fr"
                        }
                      },
                      {
                        "id": "mgZvgjUnkaG",
                        "vid": "mgZvgjUnkaG",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgrRSSGILHa",
                          "mgrRSSGILGZ"
                        ],
                        "props": {
                          "align": "center",
                          "direction": "row",
                          "display": "flex",
                          "gap": 20,
                          "height": 236.99995172502565,
                          "left": 0,
                          "position": "",
                          "top": 513.0000954185037,
                          "width": "1fr",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgrRSSGILHa",
                            "vid": "mgrRSSGILHa",
                            "hydrate_id": "",
                            "type": "frame",
                            "props": {
                              "bgColor": "rgb(230, 241, 240)",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "4px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "image",
                              "height": "237px",
                              "img": {
                                "alt": "",
                                "src": "aaf0dc348ff54ebe03a1cbbc4f60ae45/89d23d1b-be69-49dc-b564-16728aaa1f61.jpg@d_progressive",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 0,
                              "top": 513,
                              "width": "1fr"
                            }
                          },
                          {
                            "id": "mgrRSSGILGZ",
                            "vid": "mgrRSSGILGZ",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgrRSSGILGY",
                              "mgrRSSGILGX"
                            ],
                            "props": {
                              "bgColor": "rgb(230, 241, 240)",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "4px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "color",
                              "height": "237px",
                              "img": {
                                "alt": "",
                                "src": "",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 258,
                              "top": 513,
                              "width": "1fr"
                            },
                            "children": [
                              {
                                "id": "mgrRSSGILGY",
                                "vid": "mgrRSSGILGY",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 32,
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(102, 102, 102);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 28px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(102, 102, 102);border-right: 0px none rgb(102, 102, 102);border-bottom: 0px none rgb(102, 102, 102);border-left: 0px none rgb(102, 102, 102);border-radius: 0px;align-items: normal\"\u003e“希望在未来的日子里，我的理想L7能伴随着我实现理想。”\u003c/p\u003e",
                                  "top": 54
                                }
                              },
                              {
                                "id": "mgrRSSGILGX",
                                "vid": "mgrRSSGILGX",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 32,
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 21px;text-align: center;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(153, 153, 153);border-right: 0px none rgb(153, 153, 153);border-bottom: 0px none rgb(153, 153, 153);border-left: 0px none rgb(153, 153, 153);border-radius: 0px;align-items: normal\"\u003e理想汽车用户@起名想九天\u003c/p\u003e",
                                  "top": 162
                                }
                              }
                            ]
                          }
                        ]
                      },
                      {
                        "id": "mgrRSSGILGW",
                        "vid": "mgrRSSGILGW",
                        "hydrate_id": "",
                        "type": "frame",
                        "props": {
                          "bgColor": "rgb(230, 241, 240)",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "4px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "fillType": "image",
                          "height": "237px",
                          "img": {
                            "alt": "",
                            "src": "d796a76ea30b8564e9b4920a8d4c6fc5/4dcb04f8-27d0-4ad9-928a-472efe3f1a88.jpg@d_progressive",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "position": "",
                          "top": 769,
                          "width": "1fr"
                        }
                      }
                    ]
                  }
                ]
              }
            ]
          },
          {
            "id": "mfJumyOUAte",
            "vid": "mfJumyOUAte",
            "hydrate_id": "",
            "type": "frame",
            "children_ids": [
              "mawZUSixpek"
            ],
            "props": {
              "direction": "row",
              "display": "flex",
              "gap": 0,
              "height": 1348.1875469810084,
              "left": 0,
              "position": "",
              "top": 2199.9844326543,
              "width": "1fr",
              "wrap": "nowrap"
            },
            "children": [
              {
                "id": "mawZUSixpek",
                "vid": "mawZUSixpek",
                "hydrate_id": "bj",
                "type": "frame",
                "children_ids": [
                  "mawUWmKjkpv",
                  "mkeEavaPbYi",
                  "melhbTGYVyZ",
                  "mawZUSixpea"
                ],
                "props": {
                  "background": "none 0% 0% / auto repeat scroll padding-box border-box rgba(0, 0, 0, 0)",
                  "bgColor": "rgba(198,194,214,1)",
                  "borderBottom": "0px none rgb(0, 0, 0)",
                  "borderLeft": "0px none rgb(0, 0, 0)",
                  "borderRadius": "0px",
                  "borderRight": "0px none rgb(0, 0, 0)",
                  "borderTop": "0px none rgb(0, 0, 0)",
                  "direction": "column",
                  "display": "flex",
                  "fillType": "color",
                  "height": "840px",
                  "left": 0,
                  "link": "http://baidu.com",
                  "linkTarget": "_blank",
                  "pin": {
                    "top": 1,
                    "left": 1
                  },
                  "position": "",
                  "top": 508.18752824214465,
                  "width": "1fr"
                },
                "children": [
                  {
                    "id": "mawUWmKjkpv",
                    "vid": "mawUWmKjkpv",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mawZUSixpei",
                      "mawZUSixpeg"
                    ],
                    "props": {
                      "align": "center",
                      "background": "rgba(47,156,255,0)",
                      "display": "flex",
                      "distribution": "flex-start",
                      "gap": 50,
                      "height": "auto",
                      "left": 0,
                      "top": 0,
                      "width": "340px"
                    },
                    "children": [
                      {
                        "id": "mawZUSixpei",
                        "vid": "mawZUSixpei",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "position": "relative",
                          "text": "\u003ch2 style=\"opacity: 1;color: rgb(0, 0, 0);font-family: montserrat, sans-serif;font-size: 14px;font-weight: 400;line-height: normal;text-align: start;background-color: rgba(0, 0, 0, 0);background-image: none;border-top: 0px none rgb(0, 0, 0);border-right: 0px none rgb(0, 0, 0);border-bottom: 0px none rgb(0, 0, 0);border-left: 0px none rgb(0, 0, 0);border-radius: 0px;align-items: normal\"\u003e\u003cspan style=\"letter-spacing:0.16em;\" class=\"wixui-rich-text__text\"\u003e\u003cspan class=\"color_11 wixui-rich-text__text\"\u003eVISION\u003c/span\u003e\u003c/span\u003e\u003c/h2\u003e",
                          "top": 0
                        }
                      },
                      {
                        "id": "mawZUSixpeg",
                        "vid": "mawZUSixpeg__mapxGgRqUpu",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "hide-lZRRDONbpAi hide-maxgJZzNUPm",
                          "decoration": "none",
                          "fontSize": 16,
                          "left": 0,
                          "letter": "0.04em",
                          "lineHeight": "1.4em",
                          "shadow": [
                            {
                              "x": 0,
                              "y": 1,
                              "blur": 2,
                              "color": "rgba(80,227,194,1)"
                            },
                            {
                              "x": 0,
                              "y": 1,
                              "blur": 2,
                              "color": "rgba(0, 0, 0, 0.25)"
                            }
                          ],
                          "style": "3",
                          "styleId": "weave-style-1",
                          "tag": "p",
                          "text": "\u003ch1 style=\"\" class=\"weave-text weave-style-1\"\u003eWe’re Changing the Way the World Thinks About Cars\u003c/h1\u003e",
                          "top": 0
                        }
                      },
                      {
                        "id": "mawZUSixpeg",
                        "vid": "mawZUSixpeg",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "hide-mapxGgRqUpu",
                          "decoration": "none",
                          "fontSize": 16,
                          "left": 0,
                          "letter": "0.04em",
                          "lineHeight": "1.4em",
                          "shadow": [
                            {
                              "x": 0,
                              "y": 1,
                              "blur": 2,
                              "color": "rgba(80,227,194,1)"
                            },
                            {
                              "x": 0,
                              "y": 1,
                              "blur": 2,
                              "color": "rgba(0, 0, 0, 0.25)"
                            }
                          ],
                          "style": "1",
                          "styleId": "weave-style-1",
                          "tag": "h1",
                          "text": "\u003ch1 style=\"\" class=\"weave-text weave-style-1\"\u003eWe’re Changing the Way the World Thinks About Cars\u003c/h1\u003e\u003ch1 style=\"\" class=\"weave-text weave-style-1\"\u003ehahahha\u003c/h1\u003e",
                          "top": 0
                        }
                      }
                    ]
                  },
                  {
                    "id": "mkeEavaPbYi",
                    "vid": "mkeEavaPbYi",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mkeEawNZYzm"
                    ],
                    "props": {
                      "bgColor": "#1e8bfe",
                      "fillType": "color",
                      "gap": 8,
                      "height": 75.66204287515757,
                      "left": 567.9146949326573,
                      "top": 573.7294883562332,
                      "width": 65.85400028022968
                    },
                    "children": [
                      {
                        "id": "mkeEawNZYzm",
                        "vid": "mkeEawNZYzm",
                        "hydrate_id": "",
                        "type": "frame",
                        "props": {
                          "bgColor": "#2f9cff",
                          "fillType": "color",
                          "gap": 8,
                          "height": 30.825276726915945,
                          "left": 24,
                          "position": "",
                          "top": 23,
                          "width": 18.214936247723017
                        }
                      }
                    ]
                  },
                  {
                    "id": "melhbTGYVyZ",
                    "vid": "melhbTGYVyZ",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "melhbTGYVyY",
                      "melhbTGYVyX",
                      "melhbTGYVyW"
                    ],
                    "props": {
                      "bgColor": "rgba(249,225,225,1)",
                      "direction": "row",
                      "display": "",
                      "fillType": "color",
                      "gap": 68.093770990244,
                      "height": 80.58864751226349,
                      "left": -3525,
                      "pin": {
                        "top": 1,
                        "left": 1
                      },
                      "position": "relative",
                      "top": -81,
                      "width": 1599.999946533671,
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "melhbTGYVyY",
                        "vid": "melhbTGYVyY",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "height": 19.197449630624522,
                          "left": 625.1562219301773,
                          "pin": {
                            "top": 1,
                            "left": 1
                          },
                          "position": "",
                          "text": "\u003cdiv style=\"\" class=\"weave-text \"\u003eXixi da lao\u0026nbsp;\u003c/div\u003e",
                          "top": 30.6889210013602,
                          "width": 83.50134222206597
                        }
                      },
                      {
                        "id": "melhbTGYVyX",
                        "vid": "melhbTGYVyX",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "height": 19.197449630624522,
                          "left": 776.7471326007735,
                          "pin": {
                            "top": 1,
                            "left": 1
                          },
                          "position": "",
                          "text": "\u003cdiv style=\"\" class=\"weave-text \"\u003eLiang\u003c/div\u003e",
                          "top": 30.6889210013602,
                          "width": 41.40617617051829
                        }
                      },
                      {
                        "id": "melhbTGYVyW",
                        "vid": "melhbTGYVyW__mapxGgRqUpu",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "hide-lZRRDONbpAi hide-maxgJZzNUPm",
                          "_opacity": 0.5,
                          "height": 58.01379036055846,
                          "left": 1256,
                          "pin": {
                            "top": 1,
                            "left": 1
                          },
                          "position": "",
                          "text": "\u003ch1 style=\"\" class=\"weave-text-style-3ei0ki52oq1\"\u003eMake1\u003c/h1\u003e",
                          "top": 30,
                          "width": 119.71757571000717
                        }
                      },
                      {
                        "id": "melhbTGYVyW",
                        "vid": "melhbTGYVyW",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "hide-mapxGgRqUpu",
                          "height": 58.01379036055846,
                          "left": 1157,
                          "pin": {
                            "top": 1,
                            "left": 1
                          },
                          "position": "",
                          "text": "\u003ch1 style=\"\" class=\"weave-text-style-3ei0ki52oq1\"\u003eMake\u003c/h1\u003e",
                          "top": 18,
                          "width": 119.71757571000717
                        }
                      }
                    ]
                  },
                  {
                    "id": "mawZUSixpea",
                    "vid": "mawZUSixpea",
                    "hydrate_id": "",
                    "type": "text",
                    "props": {
                      "_class": "",
                      "color": "rgba(245,166,35,1)",
                      "fontSize": "18px",
                      "left": 0,
                      "letter": "1.2px",
                      "position": "",
                      "text": "\u003cp style=\"font-size: 18px;color:rgba(245,166,35,1);line-height:28.8px;letter-spacing:1.2px;text-align:start;font-family:montserrat, sans-serif;\" class=\"weave-text \"\u003e\u003cspan style=\"\"\u003eI'm a paragraph. Click here to add your own text and edit me. It’s easy. Just click “Edit Text” or double click me to add your own content and make changes to the font. I’m a great place for you to tell a story and let your users know a little more about you.\u003c/span\u003e\u003c/p\u003e",
                      "top": 505.6214222318346
                    }
                  }
                ]
              }
            ]
          },
          {
            "id": "mgykUHPtsfU",
            "vid": "mgykUHPtsfU",
            "hydrate_id": "",
            "type": "frame",
            "children_ids": [
              "mgykUHPtsfT",
              "mgykUHPtsfN",
              "mgykUHPtsfH"
            ],
            "props": {
              "align": "flex-start",
              "bgColor": "",
              "borderBottom": "0px none rgb(255, 255, 255)",
              "borderLeft": "0px none rgb(255, 255, 255)",
              "borderRadius": "0px",
              "borderRight": "0px none rgb(255, 255, 255)",
              "borderTop": "0px none rgb(255, 255, 255)",
              "direction": "row",
              "display": "flex",
              "gap": 20,
              "height": "auto",
              "img": {
                "alt": "",
                "src": "",
                "position": "center",
                "objectFit": "cover"
              },
              "left": 0,
              "padding": {
                "top": 0,
                "left": 0,
                "right": 0,
                "bottom": 0,
                "perSide": true
              },
              "position": "",
              "top": 3548.164159046987,
              "width": "1fr",
              "wrap": "wrap"
            },
            "children": [
              {
                "id": "mgykUHPtsfT",
                "vid": "mgykUHPtsfT",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mgykUHPtsfS",
                  "mgykUHPtsfR"
                ],
                "props": {
                  "align": "flex-start",
                  "bgColor": "rgb(255, 255, 255)",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "4px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "fillType": "color",
                  "gap": 0,
                  "height": "auto",
                  "img": {
                    "alt": "",
                    "src": "",
                    "position": "center",
                    "objectFit": "cover"
                  },
                  "left": 160,
                  "padding": {
                    "top": 0,
                    "left": 0,
                    "right": 0,
                    "bottom": 0,
                    "perSide": true
                  },
                  "position": "",
                  "top": 0,
                  "width": "1fr",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mgykUHPtsfS",
                    "vid": "mgykUHPtsfS",
                    "hydrate_id": "",
                    "type": "frame",
                    "props": {
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "4px 4px 0px 0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "fillType": "image",
                      "height": "279px",
                      "img": {
                        "alt": "",
                        "src": "c44271bc3a5e29f2e4f2cbea77b5aebf/740ad494-cb2b-475f-a797-41064296a47b.jpg@d_progressive",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "position": "",
                      "top": 0,
                      "width": "1fr"
                    }
                  },
                  {
                    "id": "mgykUHPtsfR",
                    "vid": "mgykUHPtsfR",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgykUHPtsfQ",
                      "mgykUHPtsfP",
                      "mgykUHPtsfO"
                    ],
                    "props": {
                      "align": "flex-start",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "column",
                      "display": "flex",
                      "gap": 4,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "top": 32,
                        "left": 32,
                        "right": 0,
                        "value": 32,
                        "bottom": 32,
                        "perSide": false
                      },
                      "position": "",
                      "top": 279,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgykUHPtsfQ",
                        "vid": "mgykUHPtsfQ",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 22px;text-align: left;align-items: normal\"\u003e理想汽车用户@卢森堡大使\u003c/p\u003e",
                          "top": 32,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgykUHPtsfP",
                        "vid": "mgykUHPtsfP",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(0, 0, 0);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: left;align-items: normal\"\u003e大家好，各位理想的车主朋友们！\u003c/p\u003e",
                          "top": 58,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgykUHPtsfO",
                        "vid": "mgykUHPtsfO",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(90, 124, 171);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 22px;text-align: left;align-items: center\"\u003e查看详情\u003c/p\u003e",
                          "top": 111,
                          "width": "74px"
                        }
                      }
                    ]
                  }
                ]
              },
              {
                "id": "mgykUHPtsfN",
                "vid": "mgykUHPtsfN",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mgykUHPtsfM",
                  "mgykUHPtsfL"
                ],
                "props": {
                  "align": "flex-start",
                  "bgColor": "rgb(255, 255, 255)",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "4px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "fillType": "color",
                  "gap": 0,
                  "height": "auto",
                  "img": {
                    "alt": "",
                    "src": "",
                    "position": "center",
                    "objectFit": "cover"
                  },
                  "left": 676,
                  "padding": {
                    "top": 0,
                    "left": 0,
                    "right": 0,
                    "bottom": 0,
                    "perSide": true
                  },
                  "position": "",
                  "top": 0,
                  "width": "1fr",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mgykUHPtsfM",
                    "vid": "mgykUHPtsfM",
                    "hydrate_id": "",
                    "type": "frame",
                    "props": {
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "4px 4px 0px 0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "fillType": "image",
                      "height": "279px",
                      "img": {
                        "alt": "",
                        "src": "b4a21fa0dd7a4f4172e6b6286aa60309/3454f570-0341-4389-b836-588a592f1b40.jpg@d_progressive",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "position": "",
                      "top": 0,
                      "width": "1fr"
                    }
                  },
                  {
                    "id": "mgykUHPtsfL",
                    "vid": "mgykUHPtsfL",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgykUHPtsfK",
                      "mgykUHPtsfJ",
                      "mgykUHPtsfI"
                    ],
                    "props": {
                      "align": "flex-start",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "column",
                      "display": "flex",
                      "gap": 4,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "top": 32,
                        "left": 32,
                        "right": 0,
                        "value": 32,
                        "bottom": "",
                        "perSide": false
                      },
                      "position": "",
                      "top": 279,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgykUHPtsfK",
                        "vid": "mgykUHPtsfK",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32.000019446343686,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 22px;text-align: left;align-items: normal\"\u003e理想汽车用户@玖肆玖\u003c/p\u003e",
                          "top": 31.999848263742933,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgykUHPtsfJ",
                        "vid": "mgykUHPtsfJ",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32.000019446343686,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(0, 0, 0);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: left;align-items: normal\"\u003e谁不想在雪山下来一片毛肚呢🏔️\u003c/p\u003e",
                          "top": 55.99999123545058,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgykUHPtsfI",
                        "vid": "mgykUHPtsfI",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(90, 124, 171);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 22px;text-align: left;align-items: center\"\u003e查看详情\u003c/p\u003e",
                          "top": 111,
                          "width": "74px"
                        }
                      }
                    ]
                  }
                ]
              },
              {
                "id": "mgykUHPtsfH",
                "vid": "mgykUHPtsfH",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mgykUHPtsfG",
                  "mgykUHPtsfF"
                ],
                "props": {
                  "align": "flex-start",
                  "bgColor": "rgb(255, 255, 255)",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "4px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "fillType": "color",
                  "gap": 0,
                  "height": "auto",
                  "img": {
                    "alt": "",
                    "src": "",
                    "position": "center",
                    "objectFit": "cover"
                  },
                  "left": 1192,
                  "padding": {
                    "top": 0,
                    "left": 0,
                    "right": 0,
                    "bottom": 0,
                    "perSide": true
                  },
                  "position": "",
                  "top": 0,
                  "width": "1fr",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mgykUHPtsfG",
                    "vid": "mgykUHPtsfG",
                    "hydrate_id": "",
                    "type": "frame",
                    "props": {
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "4px 4px 0px 0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "fillType": "image",
                      "height": "279px",
                      "img": {
                        "alt": "",
                        "src": "9950d660c41ccf6e9dd8553cf157a2f2/461872e1-a288-4f08-9926-b392ed782ea5.jpg@d_progressive",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "position": "",
                      "top": 0,
                      "width": "1fr"
                    }
                  },
                  {
                    "id": "mgykUHPtsfF",
                    "vid": "mgykUHPtsfF",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgykUHPtsfE",
                      "mgykUHPtsfD",
                      "mgykUHPtsfC"
                    ],
                    "props": {
                      "align": "flex-start",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "column",
                      "display": "flex",
                      "gap": 4,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "top": 32,
                        "left": 32,
                        "right": 0,
                        "value": 32,
                        "bottom": 32,
                        "perSide": false
                      },
                      "position": "",
                      "top": 279,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgykUHPtsfE",
                        "vid": "mgykUHPtsfE",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(153, 153, 153);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 22px;text-align: left;align-items: normal\"\u003e理想汽车用户@聂同学\u003c/p\u003e",
                          "top": 32,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgykUHPtsfD",
                        "vid": "mgykUHPtsfD",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(0, 0, 0);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: left;align-items: normal\"\u003e【疆野驰骋】理想越过大海道\u003c/p\u003e",
                          "top": 58,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgykUHPtsfC",
                        "vid": "mgykUHPtsfC",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 32,
                          "position": "",
                          "text": "\u003cp style=\"opacity: 1;color: rgb(90, 124, 171);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 14px;font-weight: 400;line-height: 22px;text-align: left;align-items: center\"\u003e查看详情\u003c/p\u003e",
                          "top": 111,
                          "width": "74px"
                        }
                      }
                    ]
                  }
                ]
              }
            ]
          },
          {
            "id": "mgymoGzJkCc",
            "vid": "mgymoGzJkCc",
            "hydrate_id": "",
            "type": "frame",
            "children_ids": [
              "mgymoGzJkCb"
            ],
            "props": {
              "align": "center",
              "bgColor": "",
              "borderBottom": "0px none rgb(255, 255, 255)",
              "borderLeft": "0px none rgb(255, 255, 255)",
              "borderRadius": "0px",
              "borderRight": "0px none rgb(255, 255, 255)",
              "borderTop": "0px none rgb(255, 255, 255)",
              "direction": "row",
              "display": "flex",
              "fillType": "image",
              "gap": 0,
              "height": "auto",
              "img": {
                "alt": "",
                "src": "12a10b3285efff727005bf7ef8fea4b0/925cddb4-716c-4157-a401-c3c8ceae7eef.jpg@d_progressive",
                "position": "center",
                "objectFit": "cover"
              },
              "left": 0,
              "padding": {
                "top": 0,
                "left": 986,
                "right": 234,
                "bottom": 0,
                "perSide": true
              },
              "position": "",
              "top": 3971.1721139709107,
              "width": "1fr",
              "wrap": "nowrap"
            },
            "children": [
              {
                "id": "mgymoGzJkCb",
                "vid": "mgymoGzJkCb",
                "hydrate_id": "",
                "type": "frame",
                "children_ids": [
                  "mhfMljJQiNu",
                  "mgymoGzJkBY"
                ],
                "props": {
                  "align": "flex-start",
                  "bgColor": "",
                  "borderBottom": "0px none rgb(255, 255, 255)",
                  "borderLeft": "0px none rgb(255, 255, 255)",
                  "borderRadius": "0px",
                  "borderRight": "0px none rgb(255, 255, 255)",
                  "borderTop": "0px none rgb(255, 255, 255)",
                  "direction": "column",
                  "display": "flex",
                  "gap": 24,
                  "height": "auto",
                  "img": {
                    "alt": "",
                    "src": "",
                    "position": "center",
                    "objectFit": "cover"
                  },
                  "left": 986,
                  "padding": {
                    "top": 68,
                    "left": 0,
                    "right": 0,
                    "bottom": 15,
                    "perSide": true
                  },
                  "position": "",
                  "top": 0,
                  "width": "auto",
                  "wrap": "nowrap"
                },
                "children": [
                  {
                    "id": "mhfMljJQiNu",
                    "vid": "mhfMljJQiNu",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgymoGzJkCa",
                      "mgymoGzJkBZ"
                    ],
                    "props": {
                      "align": "flex-start",
                      "direction": "column",
                      "display": "flex",
                      "gap": 4,
                      "height": 99.99996431344744,
                      "left": 0,
                      "position": "",
                      "top": 68.00005602788724,
                      "width": 321.9999832273204,
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgymoGzJkCa",
                        "vid": "mgymoGzJkCa",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 46px;font-weight: 400;line-height: 72px;text-align: left;align-items: normal\"\u003e理想汽车品牌书\u003c/p\u003e",
                          "top": 68,
                          "width": "auto"
                        }
                      },
                      {
                        "id": "mgymoGzJkBZ",
                        "vid": "mgymoGzJkBZ",
                        "hydrate_id": "",
                        "type": "text",
                        "props": {
                          "_class": "",
                          "left": 0,
                          "text": "\u003cp style=\"opacity: 1;color: rgb(186, 186, 186);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 24px;text-align: left;align-items: normal\"\u003e一年一册 有温度的理想“家书”\u003c/p\u003e",
                          "top": 144,
                          "width": "auto"
                        }
                      }
                    ]
                  },
                  {
                    "id": "mgymoGzJkBY",
                    "vid": "mgymoGzJkBY",
                    "hydrate_id": "",
                    "type": "frame",
                    "children_ids": [
                      "mgymoGzJkBX",
                      "mgymoGzJkBQ"
                    ],
                    "props": {
                      "align": "center",
                      "bgColor": "",
                      "borderBottom": "0px none rgb(255, 255, 255)",
                      "borderLeft": "0px none rgb(255, 255, 255)",
                      "borderRadius": "0px",
                      "borderRight": "0px none rgb(255, 255, 255)",
                      "borderTop": "0px none rgb(255, 255, 255)",
                      "direction": "row",
                      "display": "flex",
                      "gap": 40,
                      "height": "auto",
                      "img": {
                        "alt": "",
                        "src": "",
                        "position": "center",
                        "objectFit": "cover"
                      },
                      "left": 0,
                      "padding": {
                        "value": 0,
                        "perSide": false
                      },
                      "position": "",
                      "top": 192,
                      "width": "auto",
                      "wrap": "nowrap"
                    },
                    "children": [
                      {
                        "id": "mgymoGzJkBX",
                        "vid": "mgymoGzJkBX",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgymoGzJkBW",
                          "mgymoGzJkBT",
                          "mgymoGzJkBS"
                        ],
                        "props": {
                          "bgColor": "",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "0px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "height": "348px",
                          "img": {
                            "alt": "",
                            "src": "",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 0,
                          "position": "",
                          "top": 0,
                          "width": "170px"
                        },
                        "children": [
                          {
                            "id": "mgymoGzJkBW",
                            "vid": "mgymoGzJkBW",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgymoGzJkBV"
                            ],
                            "props": {
                              "bgColor": "",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "8px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "image",
                              "height": "240px",
                              "img": {
                                "alt": "",
                                "src": "cd42acb16bb17dc67d664c925a9346f2/9a49cb8b-8f5e-4284-9580-bfb18d924c5a.jpg@d_progressive",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 0,
                              "top": 0,
                              "width": "170px"
                            },
                            "children": [
                              {
                                "id": "mgymoGzJkBV",
                                "vid": "mgymoGzJkBV",
                                "hydrate_id": "",
                                "type": "frame",
                                "children_ids": [
                                  "mgymoGzJkBU"
                                ],
                                "props": {
                                  "bgColor": "",
                                  "borderBottom": "0px none rgb(255, 255, 255)",
                                  "borderLeft": "0px none rgb(255, 255, 255)",
                                  "borderRadius": "50%",
                                  "borderRight": "0px none rgb(255, 255, 255)",
                                  "borderTop": "0px none rgb(255, 255, 255)",
                                  "height": "40px",
                                  "img": {
                                    "alt": "",
                                    "src": "",
                                    "position": "center",
                                    "objectFit": "cover"
                                  },
                                  "left": 122,
                                  "top": 192,
                                  "width": "40px"
                                },
                                "children": [
                                  {
                                    "id": "mgymoGzJkBU",
                                    "vid": "mgymoGzJkBU",
                                    "hydrate_id": "",
                                    "type": "frame",
                                    "props": {
                                      "borderBottom": "0px none rgb(255, 255, 255)",
                                      "borderLeft": "0px none rgb(255, 255, 255)",
                                      "borderRadius": "0px",
                                      "borderRight": "0px none rgb(255, 255, 255)",
                                      "borderTop": "0px none rgb(255, 255, 255)",
                                      "height": "42px",
                                      "left": 6,
                                      "padding": {
                                        "value": 0,
                                        "perSide": false
                                      },
                                      "top": -1,
                                      "width": "28px"
                                    }
                                  }
                                ]
                              }
                            ]
                          },
                          {
                            "id": "mgymoGzJkBT",
                            "vid": "mgymoGzJkBT",
                            "hydrate_id": "",
                            "type": "text",
                            "props": {
                              "_class": "",
                              "left": 0,
                              "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: center;align-items: normal\"\u003e七周年品牌书\u003c/p\u003e",
                              "top": 264,
                              "width": "170px"
                            }
                          },
                          {
                            "id": "mgymoGzJkBS",
                            "vid": "mgymoGzJkBS",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgymoGzJkBR"
                            ],
                            "props": {
                              "borderBottom": "1px solid rgb(255, 255, 255)",
                              "borderLeft": "1px solid rgb(255, 255, 255)",
                              "borderRadius": "100px",
                              "borderRight": "1px solid rgb(255, 255, 255)",
                              "borderTop": "1px solid rgb(255, 255, 255)",
                              "display": "flex",
                              "height": "auto",
                              "left": 35,
                              "padding": {
                                "top": 7,
                                "left": 16,
                                "right": 16,
                                "bottom": 7,
                                "perSide": true
                              },
                              "top": 308,
                              "width": "auto"
                            },
                            "children": [
                              {
                                "id": "mgymoGzJkBR",
                                "vid": "mgymoGzJkBR",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 24px;text-align: center;align-items: normal\"\u003e开始阅读\u003c/p\u003e"
                                }
                              }
                            ]
                          }
                        ]
                      },
                      {
                        "id": "mgymoGzJkBQ",
                        "vid": "mgymoGzJkBQ",
                        "hydrate_id": "",
                        "type": "frame",
                        "children_ids": [
                          "mgymoGzJkBP",
                          "mgynzEjSpVe"
                        ],
                        "props": {
                          "bgColor": "",
                          "borderBottom": "0px none rgb(255, 255, 255)",
                          "borderLeft": "0px none rgb(255, 255, 255)",
                          "borderRadius": "0px",
                          "borderRight": "0px none rgb(255, 255, 255)",
                          "borderTop": "0px none rgb(255, 255, 255)",
                          "direction": "column",
                          "display": "flex",
                          "gap": 24,
                          "height": "348px",
                          "img": {
                            "alt": "",
                            "src": "",
                            "position": "center",
                            "objectFit": "cover"
                          },
                          "left": 210,
                          "position": "",
                          "top": 0,
                          "width": "170px",
                          "wrap": "nowrap"
                        },
                        "children": [
                          {
                            "id": "mgymoGzJkBP",
                            "vid": "mgymoGzJkBP",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgymoGzJkBO"
                            ],
                            "props": {
                              "bgColor": "",
                              "borderBottom": "0px none rgb(255, 255, 255)",
                              "borderLeft": "0px none rgb(255, 255, 255)",
                              "borderRadius": "8px",
                              "borderRight": "0px none rgb(255, 255, 255)",
                              "borderTop": "0px none rgb(255, 255, 255)",
                              "fillType": "image",
                              "height": "240px",
                              "img": {
                                "alt": "",
                                "src": "ed1de5cc8d5794e8574782b039904456/7d4a705d-984c-4d97-9db8-07e1e38b536b.jpg@d_progressive",
                                "position": "center",
                                "objectFit": "cover"
                              },
                              "left": 0,
                              "position": "",
                              "top": 0,
                              "width": "170px"
                            },
                            "children": [
                              {
                                "id": "mgymoGzJkBO",
                                "vid": "mgymoGzJkBO",
                                "hydrate_id": "",
                                "type": "frame",
                                "children_ids": [
                                  "mgymoGzJkBN"
                                ],
                                "props": {
                                  "bgColor": "",
                                  "borderBottom": "0px none rgb(255, 255, 255)",
                                  "borderLeft": "0px none rgb(255, 255, 255)",
                                  "borderRadius": "50%",
                                  "borderRight": "0px none rgb(255, 255, 255)",
                                  "borderTop": "0px none rgb(255, 255, 255)",
                                  "height": "40px",
                                  "img": {
                                    "alt": "",
                                    "src": "",
                                    "position": "center",
                                    "objectFit": "cover"
                                  },
                                  "left": 122,
                                  "top": 192,
                                  "width": "40px"
                                },
                                "children": [
                                  {
                                    "id": "mgymoGzJkBN",
                                    "vid": "mgymoGzJkBN",
                                    "hydrate_id": "",
                                    "type": "frame",
                                    "props": {
                                      "borderBottom": "0px none rgb(255, 255, 255)",
                                      "borderLeft": "0px none rgb(255, 255, 255)",
                                      "borderRadius": "0px",
                                      "borderRight": "0px none rgb(255, 255, 255)",
                                      "borderTop": "0px none rgb(255, 255, 255)",
                                      "height": "42px",
                                      "left": 6,
                                      "padding": {
                                        "value": 0,
                                        "perSide": false
                                      },
                                      "top": -1,
                                      "width": "28px"
                                    }
                                  }
                                ]
                              }
                            ]
                          },
                          {
                            "id": "mgynzEjSpVe",
                            "vid": "mgynzEjSpVe",
                            "hydrate_id": "",
                            "type": "frame",
                            "children_ids": [
                              "mgymoGzJkBM",
                              "mgymoGzJkBL"
                            ],
                            "props": {
                              "direction": "column",
                              "display": "flex",
                              "gap": 16,
                              "height": "auto",
                              "left": 0,
                              "top": 260.0001423084391,
                              "width": 169.99996967197194,
                              "wrap": "nowrap"
                            },
                            "children": [
                              {
                                "id": "mgymoGzJkBM",
                                "vid": "mgymoGzJkBM",
                                "hydrate_id": "",
                                "type": "text",
                                "props": {
                                  "_class": "",
                                  "left": 0,
                                  "position": "",
                                  "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-medium, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 20px;font-weight: 400;line-height: 28px;text-align: center;align-items: normal\"\u003e八周年品牌书\u003c/p\u003e",
                                  "top": 0,
                                  "width": "170px"
                                }
                              },
                              {
                                "id": "mgymoGzJkBL",
                                "vid": "mgymoGzJkBL",
                                "hydrate_id": "",
                                "type": "frame",
                                "children_ids": [
                                  "mgymoGzJkBK"
                                ],
                                "props": {
                                  "borderBottom": "1px solid rgb(255, 255, 255)",
                                  "borderLeft": "1px solid rgb(255, 255, 255)",
                                  "borderRadius": "100px",
                                  "borderRight": "1px solid rgb(255, 255, 255)",
                                  "borderTop": "1px solid rgb(255, 255, 255)",
                                  "display": "flex",
                                  "height": "auto",
                                  "left": 35.99995240832527,
                                  "padding": {
                                    "top": 7,
                                    "left": 16,
                                    "right": 16,
                                    "bottom": 7,
                                    "perSide": true
                                  },
                                  "position": "",
                                  "top": 47.999936544433695,
                                  "width": "auto"
                                },
                                "children": [
                                  {
                                    "id": "mgymoGzJkBK",
                                    "vid": "mgymoGzJkBK",
                                    "hydrate_id": "",
                                    "type": "text",
                                    "props": {
                                      "_class": "",
                                      "text": "\u003cp style=\"opacity: 1;color: rgb(255, 255, 255);font-family: li-regular, \u0026quot;Microsoft YaHei\u0026quot;, \u0026quot;SF Pro SC\u0026quot;, \u0026quot;SF Pro Display\u0026quot;, \u0026quot;PingFang SC\u0026quot;, \u0026quot;Segoe UI\u0026quot;, \u0026quot;Helvetica Neue\u0026quot;, Helvetica, Arial, sans-serif;font-size: 16px;font-weight: 400;line-height: 24px;text-align: center;align-items: normal\"\u003e开始阅读\u003c/p\u003e"
                                    }
                                  }
                                ]
                              }
                            ]
                          }
                        ]
                      }
                    ]
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  },
  "Lang": "en",
  "Title": "weave",
  "Style": "body{--weave-color-ct2ikxy8ypb:;}@media (prefers-color-scheme: dark) {}.weave-style-1{color: rgba(0,0,0,1);text-align: left;text-decoration: none;text-transform: none;}@media (min-width: 0px) and (max-width: 809px){.weave-style-1{font-size: 16px;line-height: 1.4em;}}@media (min-width: 810px) and (max-width: 1199px){.weave-style-1{font-size: 24px;line-height: 1.4em;}}@media (min-width: 1200px){.weave-style-1{font-size: 46px;line-height: 1.4em;}}.weave-style-2{color: #fff;text-align: left;text-decoration: none;text-transform: none;}@media (min-width: 0px){.weave-style-2{font-size: 46px;line-height: 1.4em;}}.weave-style-3{color: #333;text-align: left;text-decoration: none;text-transform: none;}@media (min-width: 0px){.weave-style-3{line-height: 1.4em;}}.weave-style-o3nqwbehcff{color: #000;text-align: left;text-decoration: none;text-transform: none;}@media (min-width: 0px){.weave-style-o3nqwbehcff{font-size: 46px;line-height: 1.4em;}}.mfQeZJeOkvC .frame-img-box{inset: 0;position: absolute;}.mgrRPCStvmc{position: relative;}.mgrRSSGILGW .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBM{position: relative;width: 170px;}.mfJtxplzrkU{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 60px;padding-left: 15px;padding-right: 15px;padding-top: 80px;position: relative;width: auto;}.mgkYWFocFui .frame-img-box{inset: 0;position: absolute;}.melhbTGYVyZ .frame-img-box{inset: 0;position: absolute;}.mgyGSXYiphK{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 10px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: auto;}.mgymoGzJkBW .frame-img-box{inset: 0;position: absolute;}.mfJtxplzrkU .frame-img-box{inset: 0;position: absolute;}.mgkYWFocFub{position: relative;}.mgykUHPtsfL .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBQ .frame-img-box{inset: 0;position: absolute;}.mgZooedCAKq .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgV{align-items: center;background-color: rgb(230, 241, 240);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 52px;height: 1frpx;justify-content: center;padding-bottom: 40px;padding-left: 22px;padding-right: 0px;padding-top: 40px;position: relative;width: 1frpx;}.mgrRPCStvma .frame-img-box{inset: 0;position: absolute;}.mgrRSSGILHb .frame-img-box{inset: 0;position: absolute;}.mgrRSSGILGY{left: 32px;position: absolute;top: 54px;}.mgykUHPtsfK{position: relative;width: auto;}.mfQeZJeOkvC{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 15px;padding-left: 15px;padding-right: 15px;padding-top: 15px;position: relative;width: 1frpx;}.mgkYWFocFuf{align-items: center;border-bottom: 0px none rgb(25, 25, 25);border-left: 0px none rgb(25, 25, 25);border-right: 0px none rgb(25, 25, 25);border-top: 0px none rgb(25, 25, 25);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 27px;padding-right: 27px;padding-top: 0px;position: relative;width: auto;}.mgrRPCStvmb{position: relative;}.mawZUSixpeg{position: relative;}.mfJumyOUAte{align-items: center;display: flex;flex-direction: row;flex-wrap: nowrap;gap: 0px;height: 1348.1875469810084px;justify-content: center;position: relative;width: 1frpx;}.mgymoGzJkBN .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBQ{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 24px;height: 348px;justify-content: center;position: relative;width: 170px;}.mfJtxplzrkS{position: relative;}.mgykUHPtsfM .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBT{left: 0px;position: absolute;top: 264px;width: 170px;}.mgymoGzJkBO{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 40px;left: 122px;position: absolute;top: 192px;width: 40px;}.mfJtxplzrlb{height: auto;position: relative;width: auto;}.mgkYQniMxVN{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: 100%;}.mawUWmKjkpv{align-items: center;display: flex;flex-direction: row;flex-wrap: wrap;gap: 50px;height: auto;justify-content: flex-start;position: relative;width: 340px;}.mgykUHPtsfH .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBS .frame-img-box{inset: 0;position: absolute;}.lZRRDONbpAi{align-items: center;background-color: rgba(255,255,255,1);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;left: -3034px;position: absolute;top: -3319px;width: 1600px;}.mgZvgjUnkaG{align-items: center;display: flex;flex-direction: row;flex-wrap: nowrap;gap: 20px;height: 236.99995172502565px;justify-content: center;position: relative;width: 1frpx;}.mgrRPCStvma{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 365px;position: relative;width: 1frpx;}.mgrRSSGILHa .frame-img-box{inset: 0;position: absolute;}.mgkYQniMxVN .frame-img-box{inset: 0;position: absolute;}.mkeEawNZYzm .frame-img-box{inset: 0;position: absolute;}.mawZUSixpek .frame-img-box{inset: 0;position: absolute;}.mawZUSixpek{align-items: center;background-color: rgba(198,194,214,1);border-bottom: 0px none rgb(0, 0, 0);border-left: 0px none rgb(0, 0, 0);border-right: 0px none rgb(0, 0, 0);border-top: 0px none rgb(0, 0, 0);display: flex;flex-direction: column;flex-wrap: wrap;height: 840px;justify-content: center;position: relative;width: 1frpx;}.mgykUHPtsfR .frame-img-box{inset: 0;position: absolute;}.mgrRPCStvmd .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBL .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfF .frame-img-box{inset: 0;position: absolute;}.mgyGixoPfWW{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 10px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: auto;}.mgZooedCAKq{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 20px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: auto;}.mgZouKFNQQy .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgP{left: 32px;position: absolute;top: 148px;}.mgymoGzJkBY{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: nowrap;gap: 40px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: auto;}.mgkXFDTNbLS{position: relative;}.mgrRPCStvme{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: nowrap;gap: 8px;height: 365px;justify-content: center;position: relative;width: 1frpx;}.melhbTGYVyZ{background-color: rgba(249,225,225,1);height: 80.58864751226349px;position: relative;width: 1599.999946533671px;}.mfJumyOUAte .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfN .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfD{position: relative;width: auto;}.mgymoGzJkBU .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkCc{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 986px;padding-right: 234px;padding-top: 0px;position: relative;width: 1frpx;}.mgrROJEgcgR .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfG{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 279px;position: relative;width: 1frpx;}.mgykUHPtsfF{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 4px;height: auto;justify-content: center;padding-bottom: 32px;padding-left: 32px;padding-right: 32px;padding-top: 32px;position: relative;width: auto;}.mgymoGzJkCa{position: relative;width: auto;}.mgykUHPtsfP{position: relative;width: auto;}.mfJtxplzrld .frame-img-box{inset: 0;position: absolute;}.mgZouKFNQQy{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 26px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: auto;}.mgrROJEgcgU{position: relative;}.mgykUHPtsfR{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 4px;height: auto;justify-content: center;padding-bottom: 32px;padding-left: 32px;padding-right: 32px;padding-top: 32px;position: relative;width: auto;}.mhfMljJQiNu .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBK{position: relative;}.mfQeWCaAMEi{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 15px;padding-left: 15px;padding-right: 15px;padding-top: 15px;position: relative;width: 1frpx;}.mgyHpplWHci{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;position: relative;width: 1frpx;}.mawUWmKjkpv .frame-img-box{inset: 0;position: absolute;}.mfJtxplzrkT{position: relative;}.mgykUHPtsfI{position: relative;width: 74px;}.mgymoGzJkBP .frame-img-box{inset: 0;position: absolute;}.mgyGixoPfWW .frame-img-box{inset: 0;position: absolute;}.mgrRSSGILGZ{background-color: rgb(230, 241, 240);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 237px;position: relative;width: 1frpx;}.mgykUHPtsfL{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 4px;height: auto;justify-content: center;padding-bottom: 32px;padding-left: 32px;padding-right: 32px;padding-top: 32px;position: relative;width: auto;}.mgynzEjSpVe{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 16px;height: auto;justify-content: center;position: relative;width: 169.99996967197194px;}.mgjuNkALlLn{align-items: center;border-bottom: 0px none rgb(25, 25, 25);border-left: 0px none rgb(25, 25, 25);border-right: 0px none rgb(25, 25, 25);border-top: 0px none rgb(25, 25, 25);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 10px;height: auto;justify-content: center;padding-bottom: 15px;padding-left: 0px;padding-right: 0px;padding-top: 15px;position: relative;width: auto;}.mfJtxplzrkV .frame-img-box{inset: 0;position: absolute;}.mgkYWFocFuf .frame-img-box{inset: 0;position: absolute;}.mgkYWFocFui{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 72px;padding-left: 0px;padding-right: 0px;padding-top: 72px;position: relative;width: auto;}.mgrROJEgcgX .frame-img-box{inset: 0;position: absolute;}.melhbTGYVyX{height: 19.197449630624522px;left: 776.7471326007735px;position: absolute;top: 30.6889210013602px;width: 41.40617617051829px;}.mgykUHPtsfQ{position: relative;width: auto;}.mgykUHPtsfT{align-items: flex-start;background-color: rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: 1frpx;}.mfJAradyRRC .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBW{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 240px;left: 0px;position: absolute;top: 0px;width: 170px;}.mgymoGzJkBS{align-items: center;border-bottom: 1px solid rgb(255, 255, 255);border-left: 1px solid rgb(255, 255, 255);border-right: 1px solid rgb(255, 255, 255);border-top: 1px solid rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: wrap;height: auto;justify-content: center;left: 35px;padding-bottom: 7px;padding-left: 16px;padding-right: 16px;padding-top: 7px;position: absolute;top: 308px;width: auto;}.mgymoGzJkBZ{position: relative;width: auto;}.mfJtxplzrkV{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex: 1 0 0;flex-direction: column;flex-wrap: nowrap;gap: 10px;height: 761px;justify-content: space-between;position: relative;}.mgrROJEgcgW{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 237px;position: relative;width: 1frpx;}.mkeEawNZYzm{background-color: #2f9cff;height: 30.825276726915945px;left: 24px;position: absolute;top: 23px;width: 18.214936247723017px;}.mgykUHPtsfT .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBN{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 42px;left: 6px;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: absolute;top: -1px;width: 28px;}.mgymoGzJkCb{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 24px;height: auto;justify-content: center;padding-bottom: 15px;padding-left: 0px;padding-right: 0px;padding-top: 68px;position: relative;width: auto;}.mhlFKZIsbSa{align-items: center;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;position: relative;width: auto;}.mfQeWCaAMEi .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgX{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex: 1 0 0;flex-direction: column;flex-wrap: nowrap;gap: 19px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;}.mgrRSSGILHc{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex: 1 0 0;flex-direction: column;flex-wrap: nowrap;gap: 19px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;}.mgymoGzJkBR{position: relative;}.mgymoGzJkBX{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 348px;position: relative;width: 170px;}.mgrROJEgcgO .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfG .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfE{position: relative;width: auto;}.mgymoGzJkCc .frame-img-box{inset: 0;position: absolute;}.mkeEavaPbYi .frame-img-box{inset: 0;position: absolute;}.mgyHtxjgrNe{align-items: center;display: flex;flex: 1 0 0;flex-direction: row;flex-wrap: nowrap;gap: 20px;justify-content: center;position: relative;width: 100%;}.melhbTGYVyY{height: 19.197449630624522px;left: 625.1562219301773px;position: absolute;top: 30.6889210013602px;width: 83.50134222206597px;}.mgykUHPtsfJ{position: relative;width: auto;}.mgymoGzJkBU{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 42px;left: 6px;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: absolute;top: -1px;width: 28px;}.mgynzEjSpVe .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgT{position: relative;}.mfJtxplzrla{height: auto;position: relative;width: auto;}.mgrROJEgcgS .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgR{background-color: rgb(230, 241, 240);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 237px;position: relative;width: 1frpx;}.mgZvdBosioO .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfS{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 279px;position: relative;width: 1frpx;}.mgykUHPtsfM{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 279px;position: relative;width: 1frpx;}.mfJAradyRRC{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 24px;padding-left: 0px;padding-right: 0px;padding-top: 24px;position: relative;width: auto;}.mgrROJEgcgO{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 237px;position: relative;width: 1frpx;}.mgrRPCStvmd{align-items: flex-start;background-color: rgb(230, 241, 240);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 24px;height: auto;justify-content: center;padding-bottom: 69px;padding-left: 32px;padding-right: 0px;padding-top: 67px;position: relative;width: 1frpx;}.mgrRSSGILHb{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 494px;position: relative;width: 1frpx;}.mgymoGzJkBV{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 40px;left: 122px;position: absolute;top: 192px;width: 40px;}.mgkYWFocFud{position: relative;}.mfJtxplzrkY{height: auto;position: relative;width: auto;}.mfJtxplzrkO{position: relative;width: auto;}.mgZvdBosioO{align-items: center;display: flex;flex-direction: row;flex-wrap: nowrap;gap: 20px;height: 236.99995172502577px;justify-content: center;position: relative;width: 1frpx;}.mgrRPCStvme .frame-img-box{inset: 0;position: absolute;}.melhbTGYVyW{height: 58.01379036055846px;left: 1157px;position: absolute;top: 18px;width: 119.71757571000717px;}.mawZUSixpea{position: relative;}.mgykUHPtsfU .frame-img-box{inset: 0;position: absolute;}.mgjuNkALlLm{position: relative;}.mgrRSSGILGX{left: 32px;position: absolute;top: 162px;}.mgkZaBDydTS{align-items: center;display: flex;flex-direction: row;flex-wrap: wrap;gap: 12px;height: auto;justify-content: center;position: relative;width: 100%;}.mawZUSixpei{position: relative;}.mgykUHPtsfS .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfN{align-items: flex-start;background-color: rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: 1frpx;}.mgymoGzJkBY .frame-img-box{inset: 0;position: absolute;}.mhlFKZIsbSa .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgS{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 494px;position: relative;width: 100%;}.mgrROJEgcgQ{left: 32px;position: absolute;top: 68px;}.mkeEavaPbYi{background-color: #1e8bfe;height: 75.66204287515757px;position: relative;width: 65.85400028022968px;}.mgykUHPtsfU{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: wrap;gap: 20px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: 1frpx;}.mgymoGzJkBV .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBX .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkBO .frame-img-box{inset: 0;position: absolute;}.mfJtxplzrle .frame-img-box{inset: 0;position: absolute;}.mgrRPCStvmf .frame-img-box{inset: 0;position: absolute;}.mgrRSSGILHa{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 237px;position: relative;width: 1frpx;}.mgZvgjUnkaG .frame-img-box{inset: 0;position: absolute;}.mgykUHPtsfH{align-items: flex-start;background-color: rgb(255, 255, 255);border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: column;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;width: 1frpx;}.mhfMljJQiNu{align-items: flex-start;display: flex;flex-direction: column;flex-wrap: nowrap;gap: 4px;height: 99.99996431344744px;justify-content: center;position: relative;width: 321.9999832273204px;}.mgymoGzJkBP{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 240px;position: relative;width: 170px;}.mgymoGzJkBL{align-items: center;border-bottom: 1px solid rgb(255, 255, 255);border-left: 1px solid rgb(255, 255, 255);border-right: 1px solid rgb(255, 255, 255);border-top: 1px solid rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: wrap;height: auto;justify-content: center;padding-bottom: 7px;padding-left: 16px;padding-right: 16px;padding-top: 7px;position: relative;width: auto;}.mgyHtxjgrNe .frame-img-box{inset: 0;position: absolute;}.mfJtxplzrle{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: nowrap;gap: 20.000056216591133px;height: 1frpx;justify-content: center;max-width: 1400px;position: relative;width: 1frpx;}.mgrROJEgcgW .frame-img-box{inset: 0;position: absolute;}.mgrRPCStvmf{align-items: flex-start;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex: 1 0 0;flex-direction: column;flex-wrap: nowrap;gap: 20px;height: auto;justify-content: center;padding-bottom: 0px;padding-left: 0px;padding-right: 0px;padding-top: 0px;position: relative;}.mgrRSSGILGW{border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);height: 237px;position: relative;width: 1frpx;}.mfJtxplzrld{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex: 1 0 0;flex-direction: column;flex-wrap: nowrap;gap: 10px;height: 761px;justify-content: space-between;position: relative;}.mgyGSXYiphK .frame-img-box{inset: 0;position: absolute;}.mgyHpplWHci .frame-img-box{inset: 0;position: absolute;}.mgrROJEgcgV .frame-img-box{inset: 0;position: absolute;}.mgrRSSGILGZ .frame-img-box{inset: 0;position: absolute;}.mgrRSSGILHc .frame-img-box{inset: 0;position: absolute;}.mgkZaBDydTS .frame-img-box{inset: 0;position: absolute;}.mgymoGzJkCb .frame-img-box{inset: 0;position: absolute;}.mfJtxplzrlc .frame-img-box{inset: 0;position: absolute;}.lZRRDONbpAi .frame-img-box{inset: 0;position: absolute;}.mfJtxplzrlc{align-items: center;border-bottom: 0px none rgb(255, 255, 255);border-left: 0px none rgb(255, 255, 255);border-right: 0px none rgb(255, 255, 255);border-top: 0px none rgb(255, 255, 255);display: flex;flex-direction: row;flex-wrap: nowrap;gap: 0px;height: auto;justify-content: center;padding-bottom: 15px;padding-left: 15px;padding-right: 15px;padding-top: 80px;position: relative;width: auto;}.mfJtxplzrkW{position: relative;}.mfJtxplzrkQ{position: relative;}.mgykUHPtsfO{position: relative;width: 74px;}.mgykUHPtsfC{position: relative;width: 74px;}.mgjuNkALlLn .frame-img-box{inset: 0;position: absolute;}@media (max-width: 400px){.mgymoGzJkCb{height: 539.9917336130343px;padding-bottom: 0px;width: 379.9998853523856px;}.melhbTGYVyW{left: 1120px;}.mgynzEjSpVe{height: 88.00027248566585px;width: 170.00026128762534px;}.mfJtxplzrld{gap: 150px;height: 800px;width: 100%;}.mhfMljJQiNu{height: 99.99996431344789px;width: 321.99976018636943px;}.mgymoGzJkBY{gap: 20px;justify-content: space-between;width: 1frpx;}.mfQeWCaAMEi{width: 193.50804182867452px;}.mhlFKZIsbSa{height: 8293.11715618167px;width: 399.99973901390905px;}.melhbTGYVyY{left: 199.67958908672483px;top: 30.695328500759786px;}.mfJtxplzrle{flex-direction: column;}.mgrRSSGILHc{width: 100%;}.mfJtxplzrkV{gap: 150px;height: auto;width: 100%;}.melhbTGYVyX{left: 15.328039662614515px;top: 30.695328500759786px;}.lZRRDONbpAi{left: 332px;top: -3214px;width: 400px;}.mgkZaBDydTS{flex-direction: column;}.mgykUHPtsfU{flex-direction: column;}.mgrRPCStvmf{width: 100%;}.mgyGixoPfWW{height: 209.9843270407515px;width: 370.00012862109435px;}.mgZouKFNQQy{height: 182.00004570153774px;width: 249.00770194059135px;}.mgZvgjUnkaG{height: 236.99995172502577px;}.mgrROJEgcgX{width: 100%;}.mgymoGzJkCc{padding-bottom: 15px;padding-left: 15px;padding-right: 15px;}}@media (max-width: 1200px) and (min-width: 401px){.melhbTGYVyW{left: 1256px;top: 30px;}.lZRRDONbpAi{left: -1261px;top: -3102px;width: 1200px;}.mgymoGzJkCc{padding-left: 0px;padding-right: 0px;}.mgrRPCStvmf{flex: 50% 0 0;}.mhlFKZIsbSa{height: 6547.148382930181px;width: 1200.000106767037px;}.mgrROJEgcgX{flex: 51% 0 0;}.mgynzEjSpVe{height: 87.99998087001359px;width: 169.99996967197202px;}.mhfMljJQiNu{height: 99.99996431344698px;width: 321.99998322732046px;}.mgrRSSGILHc{flex: 50% 0 0;}}.v-lZRRDONbpAi .hide-lZRRDONbpAi, .v-mapxGgRqUpu .hide-mapxGgRqUpu, .v-maxgJZzNUPm .hide-maxgJZzNUPm{display: none;}\n@media (max-width: 400px){.hide-maxgJZzNUPm{display: none;}}\n@media (max-width: 1200px) and (min-width: 401px){.hide-mapxGgRqUpu{display: none;}}\n@media (min-width: 1201px){.hide-lZRRDONbpAi{display: none;}}\n",
  "FsDomain": "https://fs.huglight.cn",
  "HydrateJson": "{\"mount\":{\"bj\":[{\"type\":\"link\",\"data\":{\"link\":\"http://baidu.com\",\"linkSection\":\"\",\"linkType\":\"custom\",\"newTarget\":false,\"scrollSmooth\":false}}]},\"variantCss\":{\".lZRRDONbpAi\":[{\"css\":{\"background-color\":\"rgba(255,255,255,1)\",\"height\":\"auto\",\"left\":\"-3034px\",\"top\":\"-3319px\",\"width\":\"1600px\"},\"variant\":\"\"}],\".mawUWmKjkpv\":[{\"css\":{\"height\":\"auto\",\"width\":\"340px\"},\"variant\":\"\"}],\".mawZUSixpek\":[{\"css\":{\"background-color\":\"rgba(198,194,214,1)\",\"height\":\"840px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".melhbTGYVyW\":[{\"css\":{\"height\":\"58.01379036055846px\",\"left\":\"1157px\",\"top\":\"18px\",\"width\":\"119.71757571000717px\"},\"variant\":\"\"}],\".melhbTGYVyX\":[{\"css\":{\"height\":\"19.197449630624522px\",\"left\":\"776.7471326007735px\",\"top\":\"30.6889210013602px\",\"width\":\"41.40617617051829px\"},\"variant\":\"\"}],\".melhbTGYVyY\":[{\"css\":{\"height\":\"19.197449630624522px\",\"left\":\"625.1562219301773px\",\"top\":\"30.6889210013602px\",\"width\":\"83.50134222206597px\"},\"variant\":\"\"}],\".melhbTGYVyZ\":[{\"css\":{\"background-color\":\"rgba(249,225,225,1)\",\"height\":\"80.58864751226349px\",\"width\":\"1599.999946533671px\"},\"variant\":\"\"}],\".mfJAradyRRC\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrkO\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrkU\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrkV\":[{\"css\":{\"height\":\"761px\"},\"variant\":\"\"}],\".mfJtxplzrkY\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrla\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrlb\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrlc\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mfJtxplzrld\":[{\"css\":{\"height\":\"761px\"},\"variant\":\"\"}],\".mfJtxplzrle\":[{\"css\":{\"height\":\"1frpx\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mfJumyOUAte\":[{\"css\":{\"height\":\"1348.1875469810084px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mfQeWCaAMEi\":[{\"css\":{\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mfQeZJeOkvC\":[{\"css\":{\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgZooedCAKq\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgZouKFNQQy\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgZvdBosioO\":[{\"css\":{\"height\":\"236.99995172502577px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgZvgjUnkaG\":[{\"css\":{\"height\":\"236.99995172502565px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgjuNkALlLn\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgkYQniMxVN\":[{\"css\":{\"height\":\"auto\",\"width\":\"100%\"},\"variant\":\"\"}],\".mgkYWFocFuf\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgkYWFocFui\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgkZaBDydTS\":[{\"css\":{\"height\":\"auto\",\"width\":\"100%\"},\"variant\":\"\"}],\".mgrROJEgcgO\":[{\"css\":{\"height\":\"237px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrROJEgcgP\":[{\"css\":{\"left\":\"32px\",\"top\":\"148px\"},\"variant\":\"\"}],\".mgrROJEgcgQ\":[{\"css\":{\"left\":\"32px\",\"top\":\"68px\"},\"variant\":\"\"}],\".mgrROJEgcgR\":[{\"css\":{\"background-color\":\"rgb(230, 241, 240)\",\"height\":\"237px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrROJEgcgS\":[{\"css\":{\"height\":\"494px\",\"width\":\"100%\"},\"variant\":\"\"}],\".mgrROJEgcgV\":[{\"css\":{\"background-color\":\"rgb(230, 241, 240)\",\"height\":\"1frpx\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrROJEgcgW\":[{\"css\":{\"height\":\"237px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrROJEgcgX\":[{\"css\":{\"height\":\"auto\"},\"variant\":\"\"}],\".mgrRPCStvma\":[{\"css\":{\"height\":\"365px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRPCStvmd\":[{\"css\":{\"background-color\":\"rgb(230, 241, 240)\",\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRPCStvme\":[{\"css\":{\"height\":\"365px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRPCStvmf\":[{\"css\":{\"height\":\"auto\"},\"variant\":\"\"}],\".mgrRSSGILGW\":[{\"css\":{\"height\":\"237px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRSSGILGX\":[{\"css\":{\"left\":\"32px\",\"top\":\"162px\"},\"variant\":\"\"}],\".mgrRSSGILGY\":[{\"css\":{\"left\":\"32px\",\"top\":\"54px\"},\"variant\":\"\"}],\".mgrRSSGILGZ\":[{\"css\":{\"background-color\":\"rgb(230, 241, 240)\",\"height\":\"237px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRSSGILHa\":[{\"css\":{\"height\":\"237px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRSSGILHb\":[{\"css\":{\"height\":\"494px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgrRSSGILHc\":[{\"css\":{\"height\":\"auto\"},\"variant\":\"\"}],\".mgyGSXYiphK\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgyGixoPfWW\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgyHpplWHci\":[{\"css\":{\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgyHtxjgrNe\":[{\"css\":{\"width\":\"100%\"},\"variant\":\"\"}],\".mgykUHPtsfC\":[{\"css\":{\"width\":\"74px\"},\"variant\":\"\"}],\".mgykUHPtsfD\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfE\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfF\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfG\":[{\"css\":{\"height\":\"279px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgykUHPtsfH\":[{\"css\":{\"background-color\":\"rgb(255, 255, 255)\",\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgykUHPtsfI\":[{\"css\":{\"width\":\"74px\"},\"variant\":\"\"}],\".mgykUHPtsfJ\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfK\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfL\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfM\":[{\"css\":{\"height\":\"279px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgykUHPtsfN\":[{\"css\":{\"background-color\":\"rgb(255, 255, 255)\",\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgykUHPtsfO\":[{\"css\":{\"width\":\"74px\"},\"variant\":\"\"}],\".mgykUHPtsfP\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfQ\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfR\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgykUHPtsfS\":[{\"css\":{\"height\":\"279px\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgykUHPtsfT\":[{\"css\":{\"background-color\":\"rgb(255, 255, 255)\",\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgykUHPtsfU\":[{\"css\":{\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgymoGzJkBL\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgymoGzJkBM\":[{\"css\":{\"width\":\"170px\"},\"variant\":\"\"}],\".mgymoGzJkBN\":[{\"css\":{\"height\":\"42px\",\"left\":\"6px\",\"top\":\"-1px\",\"width\":\"28px\"},\"variant\":\"\"}],\".mgymoGzJkBO\":[{\"css\":{\"height\":\"40px\",\"left\":\"122px\",\"top\":\"192px\",\"width\":\"40px\"},\"variant\":\"\"}],\".mgymoGzJkBP\":[{\"css\":{\"height\":\"240px\",\"width\":\"170px\"},\"variant\":\"\"}],\".mgymoGzJkBQ\":[{\"css\":{\"height\":\"348px\",\"width\":\"170px\"},\"variant\":\"\"}],\".mgymoGzJkBS\":[{\"css\":{\"height\":\"auto\",\"left\":\"35px\",\"top\":\"308px\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgymoGzJkBT\":[{\"css\":{\"left\":\"0px\",\"top\":\"264px\",\"width\":\"170px\"},\"variant\":\"\"}],\".mgymoGzJkBU\":[{\"css\":{\"height\":\"42px\",\"left\":\"6px\",\"top\":\"-1px\",\"width\":\"28px\"},\"variant\":\"\"}],\".mgymoGzJkBV\":[{\"css\":{\"height\":\"40px\",\"left\":\"122px\",\"top\":\"192px\",\"width\":\"40px\"},\"variant\":\"\"}],\".mgymoGzJkBW\":[{\"css\":{\"height\":\"240px\",\"left\":\"0px\",\"top\":\"0px\",\"width\":\"170px\"},\"variant\":\"\"}],\".mgymoGzJkBX\":[{\"css\":{\"height\":\"348px\",\"width\":\"170px\"},\"variant\":\"\"}],\".mgymoGzJkBY\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgymoGzJkBZ\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgymoGzJkCa\":[{\"css\":{\"width\":\"auto\"},\"variant\":\"\"}],\".mgymoGzJkCb\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mgymoGzJkCc\":[{\"css\":{\"height\":\"auto\",\"width\":\"1frpx\"},\"variant\":\"\"}],\".mgynzEjSpVe\":[{\"css\":{\"height\":\"auto\",\"width\":\"169.99996967197194px\"},\"variant\":\"\"}],\".mhfMljJQiNu\":[{\"css\":{\"height\":\"99.99996431344744px\",\"width\":\"321.9999832273204px\"},\"variant\":\"\"}],\".mhlFKZIsbSa\":[{\"css\":{\"height\":\"auto\",\"width\":\"auto\"},\"variant\":\"\"}],\".mkeEavaPbYi\":[{\"css\":{\"background-color\":\"#1e8bfe\",\"height\":\"75.66204287515757px\",\"width\":\"65.85400028022968px\"},\"variant\":\"\"}],\".mkeEawNZYzm\":[{\"css\":{\"background-color\":\"#2f9cff\",\"height\":\"30.825276726915945px\",\"left\":\"24px\",\"top\":\"23px\",\"width\":\"18.214936247723017px\"},\"variant\":\"\"}]}}",
  "PageId": "lZRRBAQLehu",
  "SiteId": "b"
}`
