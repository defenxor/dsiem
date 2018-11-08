// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package ossimcnv

import (
	"errors"
	"os"
	"path"

	"github.com/defenxor/dsiem/internal/pkg/shared/fs"

	"github.com/dogenzaka/tsv"
)

const (
	ossimTaxonomyTSV      = "ossim_alarm_taxonomy.tsv"
	ossimKingdomsTSV      = "ossim_alarm_kingdom.tsv"
	ossimCategoriesTSV    = "ossim_alarm_category.tsv"
	ossimProductTSV       = "ossim_product_type.tsv"
	ossimProductCatTSV    = "ossim_product_category.tsv"
	ossimProductSubCatTSV = "ossim_product_subcategory.tsv"
)

// for alarm based
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

// for product based

type product struct {
	ID   int    `tsv:"id"`
	Name string `tsv:"name"`
}

type productCategory struct {
	ID   int    `tsv:"id"`
	Name string `tsv:"name"`
}

type productSubCategory struct {
	ID    int    `tsv:"id"`
	CatID int    `tsv:"cat_id"`
	Name  string `tsv:"name"`
}

type products struct {
	Products []product
}
type pcategories struct {
	Categories []productCategory
}
type psubcategories struct {
	SubCategories []productSubCategory
}

var oProd products
var oPcat pcategories
var oPsub psubcategories

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

// ParseOSSIMTSVs parses table dumps from OSSIM DB to obtain required lookup vars
func ParseOSSIMTSVs(ossimRefDir string) error {
	fTaxo := path.Join(ossimRefDir, ossimTaxonomyTSV)
	fCat := path.Join(ossimRefDir, ossimCategoriesTSV)
	fKing := path.Join(ossimRefDir, ossimKingdomsTSV)
	fProd := path.Join(ossimRefDir, ossimProductTSV)
	fPcat := path.Join(ossimRefDir, ossimProductCatTSV)
	fPsub := path.Join(ossimRefDir, ossimProductSubCatTSV)

	if !fs.FileExist(fProd) {
		return errors.New(fProd + " doesnt exist.")
	}
	if !fs.FileExist(fPcat) {
		return errors.New(fPcat + " doesnt exist.")
	}
	if !fs.FileExist(fPsub) {
		return errors.New(fPsub + " doesnt exist.")
	}
	if !fs.FileExist(fTaxo) {
		return errors.New(fTaxo + " doesnt exist.")
	}
	if !fs.FileExist(fKing) {
		return errors.New(fKing + " doesnt exist.")
	}
	if !fs.FileExist(fCat) {
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
	f4, err := os.Open(fProd)
	if err != nil {
		return err
	}
	defer f4.Close()
	f5, err := os.Open(fPcat)
	if err != nil {
		return err
	}
	defer f5.Close()
	f6, err := os.Open(fPsub)
	if err != nil {
		return err
	}
	defer f6.Close()

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

	d2 := category{}
	parser, _ = tsv.NewParser(f2, &d2)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oCat.Categories = append(oCat.Categories, d2)
		if eof {
			break
		}
	}

	d3 := kingdom{}
	parser, _ = tsv.NewParser(f3, &d3)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oKing.Kingdoms = append(oKing.Kingdoms, d3)
		if eof {
			break
		}
	}

	d4 := product{}
	parser, _ = tsv.NewParser(f4, &d4)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oProd.Products = append(oProd.Products, d4)
		if eof {
			break
		}
	}

	d5 := productCategory{}
	parser, _ = tsv.NewParser(f5, &d5)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oPcat.Categories = append(oPcat.Categories, d5)
		if eof {
			break
		}
	}

	d6 := productSubCategory{}
	parser, _ = tsv.NewParser(f6, &d6)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		oPsub.SubCategories = append(oPsub.SubCategories, d6)
		if eof {
			break
		}
	}

	return nil
}
