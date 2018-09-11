package main

import (
	"errors"
	"os"
	"path"

	"github.com/dogenzaka/tsv"
)

const (
	ossimRefDir        = "ossimref"
	ossimTaxonomyTSV   = "ossim_taxonomy.tsv"
	ossimKingdomsTSV   = "ossim_kingdoms.tsv"
	ossimCategoriesTSV = "ossim_categories.tsv"
)

type category struct {
	Name string `tsv:"name"`
	ID   int    `tsv:"id"`
}
type kingdom struct {
	Name string `tsv:"name"`
	ID   int    `tsv:"id"`
}
type taxonomy struct {
	SID      int `tsv:"sid"`
	Kingdom  int `tsv:"kingdom"`
	Category int `tsv:"category"`
}
type categories struct {
	Categories []category
}
type kingdoms struct {
	Kingdoms []kingdom
}
type taxonomies struct {
	Taxonomies []taxonomy
}

var oCat categories
var oKing kingdoms
var oTaxo taxonomies

func findKingdomCategory(sid int) (kingdom string, category string) {
	for i := range oTaxo.Taxonomies {
		if oTaxo.Taxonomies[i].SID == sid {
			nk := oTaxo.Taxonomies[i].Kingdom
			nc := oTaxo.Taxonomies[i].Category
			for j := range oCat.Categories {
				if oCat.Categories[j].ID == nc {
					category = oCat.Categories[j].Name
				}
			}
			for j := range oKing.Kingdoms {
				if oKing.Kingdoms[j].ID == nk {
					kingdom = oKing.Kingdoms[j].Name
				}
			}
		}
	}
	return kingdom, category
}

func parseOSSIMTSVs() error {
	fTaxo := path.Join(progDir, ossimRefDir, ossimTaxonomyTSV)
	fCat := path.Join(progDir, ossimRefDir, ossimCategoriesTSV)
	fKing := path.Join(progDir, ossimRefDir, ossimKingdomsTSV)
	if !fileExist(fTaxo) {
		return errors.New(fTaxo + " doesnt exist.")
	}
	if !fileExist(fKing) {
		return errors.New(fKing + " doesnt exist.")
	}
	if !fileExist(fCat) {
		return errors.New(fCat + " doesnt exist.")
	}
	f1, err := os.Open(fTaxo)
	if err != nil {
		return err
	}
	defer f1.Close()
	f2, err := os.Open(fCat)
	if err != nil {
		return err
	}
	defer f2.Close()
	f3, err := os.Open(fKing)
	if err != nil {
		return err
	}
	defer f3.Close()

	d1 := taxonomy{}
	parser, _ := tsv.NewParser(f1, &d1)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oTaxo.Taxonomies = append(oTaxo.Taxonomies, d1)
		if eof {
			break
		}
	}

	d2 := kingdom{}
	parser, _ = tsv.NewParser(f2, &d2)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oKing.Kingdoms = append(oKing.Kingdoms, d2)
		if eof {
			break
		}
	}

	d3 := category{}
	parser, _ = tsv.NewParser(f3, &d3)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oCat.Categories = append(oCat.Categories, d3)
		if eof {
			break
		}
	}

	return nil
}
