package textseg

//go:generate go run make_tables.go -output tables.go
//go:generate go run make_test_tables.go -output tables_test.go
//go:generate ruby unicode2ragel.rb --url=http://www.unicode.org/Public/9.0.0/ucd/auxiliary/GraphemeBreakProperty.txt -m GraphemeCluster -p "Prepend,CR,LF,Control,Extend,Regional_Indicator,SpacingMark,L,V,T,LV,LVT,E_Base,E_Modifier,ZWJ,Glue_After_Zwj,E_Base_GAZ" -o grapheme_clusters_table.rl
//go:generate ragel -Z grapheme_clusters.rl
//go:generate gofmt -w grapheme_clusters.go
