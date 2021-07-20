package main

import (
	"os"

	"github.com/michaelhelvey/docspec"
)

// renderer test
func main() {
	DocSpecExample()
}

func DocSpecExample() {
	var err error

	renderer, err := docspec.NewPDFRenderer(
		docspec.DocumentSizeLetter,
		"./example/fonts",
		docspec.FontConfig{Name: "Inter", Style: "", File: "Inter-Regular.json"},
		docspec.FontConfig{Name: "Inter", Style: "B", File: "Inter-Bold.json"},
		docspec.FontConfig{Name: "Inter", Style: "I", File: "Inter-Italic.json"},
	)

	if err != nil {
		panic(err)
	}

	builder := docspec.NewDocumentBuilder(renderer, docspec.DocumentSizeLetter, 5.0)

	nodeList := docspec.CreateNodeList(
		docspec.Div(
			nil,
			docspec.LayoutNodeProps{
				Border:  docspec.NewSingletonBooleanQuad(true),
				Padding: docspec.NewSingletonSizeQuad(5.0),
				Width:   docspec.WidthFill(),
				Height:  docspec.StaticSize(200),
			},
			func(parent *docspec.LayoutNode) {
				docspec.Div(parent,
					docspec.LayoutNodeProps{
						Border:      docspec.NewSingletonBooleanQuad(true),
						BorderColor: docspec.NewColor(255, 255, 255),
						Padding:     docspec.NewSingletonSizeQuad(5.0),
						Width:       docspec.WidthPercentage(60),
						Height:      docspec.StaticSize(50.0),
					},
					func(parent *docspec.LayoutNode) {
						docspec.Text(
							parent,
							docspec.LayoutNodeProps{
								Border:      docspec.NewSingletonBooleanQuad(true),
								BorderColor: docspec.NewColor(0, 255, 0),
								Padding:     docspec.NewSingletonSizeQuad(5.0),
								Width:       docspec.WidthFill(),
								Height:      docspec.HeightFill(),
							},
							docspec.TextNode{
								Text:       `Hello, World!  We can wrap text and everything, how about that!`,
								FontSize:   12.0,
								LineHeight: 1.3,
							},
						)
					},
				)
				docspec.Div(
					parent,
					docspec.LayoutNodeProps{
						Border:             docspec.NewSingletonBooleanQuad(true),
						Padding:            docspec.NewSingletonSizeQuad(5.0),
						Width:              docspec.WidthFill(),
						Height:             docspec.StaticSize(50),
						Margin:             docspec.NewSizeQuad(5, 0, 5, 0),
						ChildFlowDirection: docspec.FlowHorizontal,
						ChildAlignment:     docspec.LayoutChildAlignment{Horizontal: docspec.Center, Vertical: docspec.Center},
					},
					func(parent *docspec.LayoutNode) {
						docspec.Div(
							parent,
							docspec.LayoutNodeProps{
								Border:  docspec.NewSingletonBooleanQuad(true),
								Padding: docspec.NewSingletonSizeQuad(5.0),
								Width:   docspec.WidthPercentage(40),
								Height:  docspec.HeightFill(),
								Margin:  docspec.NewSizeQuad(0.0, 10.0, 0.0, 0.0),
							},
							func(parent *docspec.LayoutNode) {
								docspec.Image(
									parent,
									docspec.LayoutNodeProps{
										Width:  docspec.WidthFill(),
										Height: docspec.HeightFill(),
									},
									docspec.ImageNode{
										Src:           "./example/example_image.jpg",
										RatioBehavior: docspec.ImagePreserve,
										Alignment:     docspec.ImageCenter,
									},
								)
							},
						)
						docspec.Div(
							parent,
							docspec.LayoutNodeProps{
								Border:  docspec.NewSingletonBooleanQuad(true),
								Padding: docspec.NewSingletonSizeQuad(5.0),
								Width:   docspec.WidthPercentage(40),
								Height:  docspec.HeightFill(),
							},
							docspec.NoChildren,
						)
					},
				)
				docspec.Div(
					parent,
					docspec.LayoutNodeProps{
						Border:             docspec.NewSingletonBooleanQuad(true),
						Padding:            docspec.NewSingletonSizeQuad(5.0),
						Width:              docspec.WidthAsChildren(),
						Height:             docspec.StaticSize(50),
						ChildFlowDirection: docspec.FlowHorizontal,
						ChildAlignment:     docspec.LayoutChildAlignment{Horizontal: docspec.End, Vertical: docspec.Start},
					},
					func(parent *docspec.LayoutNode) {
						docspec.Div(
							parent,
							docspec.LayoutNodeProps{
								Border:  docspec.NewSingletonBooleanQuad(true),
								Padding: docspec.NewSingletonSizeQuad(5.0),
								Width:   docspec.WidthAsChildren(),
								Height:  docspec.HeightFill(),
							},
							func(parent *docspec.LayoutNode) {
								docspec.Text(
									parent,
									docspec.LayoutNodeProps{
										Border:      docspec.NewSingletonBooleanQuad(true),
										BorderColor: docspec.NewColor(0, 255, 0),
										Padding:     docspec.NewSingletonSizeQuad(5.0),
										Width:       docspec.WidthAsChildren(),
										Height:      docspec.HeightAsChildren(),
									},
									docspec.TextNode{
										Text:       `Here is some long text that should have inherent width and height`,
										FontSize:   12.0,
										LineHeight: 1.3,
									},
								)
							},
						)
					},
				)
			},
		),
	)

	err = builder.CreateDocumentTree(nodeList)

	if err != nil {
		panic(err)
	}

	builder.PrettyPrintDocumentTree()
	file, err := os.Create("./example/renderer_test.pdf")

	defer file.Close()
	if err != nil {
		panic(err)
	}

	err = builder.RenderToWriter(file)
	if err != nil {
		panic(err)
	}
}
