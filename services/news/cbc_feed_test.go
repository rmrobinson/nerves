package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type cbcDescriptionToArticleTest struct {
	name   string
	desc   string
	result *Article
}

var cbcDescriptionToArticleTests = []cbcDescriptionToArticleTest{
	{
		"basic article description",
		`
                        <img src='https://i.cbc.ca/1.4973610.1547151575!/fileImage/httpImage/image.JPG_gen/derivatives/16x9_460/maduro.JPG' alt='Maduro' width='460' title='President Maduro poses Thursday morning at the presidential palace with the Latin American leaders who joined him for his second inauguration. Left to right, Salvador Sanchez Ceren of El Salvador, Evo Morales of Bolivia, Nicolas Maduro, and Miguel Diaz-Canel of Cuba.' height='259' />                <p>Canada has declared Venezuela is now "fully entrenched in dictatorship" and says it will now recognize opposition figure Juan Guaido as the legitimate authority in the country. GAC officials expect retaliation and are preparing to have to evacuate diplomats from the country as soon as tomorrow.</p>
        `,
		&Article{
			Image: &Image{
				Name:   "i.cbc.ca/1.4973610.1547151575!/fileImage/httpImage/image.JPG_gen/derivatives/16x9_460/maduro.JPG",
				Link:   "https://i.cbc.ca/1.4973610.1547151575!/fileImage/httpImage/image.JPG_gen/derivatives/16x9_460/maduro.JPG",
				Width:  460,
				Height: 259,
				Title:  `President Maduro poses Thursday morning at the presidential palace with the Latin American leaders who joined him for his second inauguration. Left to right, Salvador Sanchez Ceren of El Salvador, Evo Morales of Bolivia, Nicolas Maduro, and Miguel Diaz-Canel of Cuba.`,
			},
			Description: `Canada has declared Venezuela is now "fully entrenched in dictatorship" and says it will now recognize opposition figure Juan Guaido as the legitimate authority in the country. GAC officials expect retaliation and are preparing to have to evacuate diplomats from the country as soon as tomorrow.`,
		},
	},
}

func TestCBCParseDescription(t *testing.T) {
	cbcf := NewCBCFeed(zaptest.NewLogger(t), "", nil)
	for _, tt := range cbcDescriptionToArticleTests {
		t.Run(tt.name, func(t *testing.T) {
			res := cbcf.parseDescription(tt.desc)
			assert.Equal(t, tt.result, res)
		})
	}
}
