// Package xmlpath implements a strict subset of the XPath specification for the Go language.
//
// The XPath specification is available at:
//
//     http://www.w3.org/TR/xpath
//
// Path expressions supported by this package are in the following format,
// with all components being optional:
//
//     /axis-name::node-test[predicate]/axis-name::node-test[predicate]
//
// At the moment, xmlpath is compatible with the XPath specification
// to the following extent:
//
//     - All axes are supported ("child", "following-sibling", etc)
//     - All abbreviated forms are supported (".", "//", etc)
//     - All node types except for namespace are supported
//     - Predicates are restricted to [N], [path], and [path=literal] forms
//     - Only a single predicate is supported per path step
//     - Namespaces are experimentally supported
//     - Richer expressions
//
// For example, assuming the following document:
//
//     <library>
//       <!-- Great book. -->
//       <book id="b0836217462" available="true">
//         <isbn>0836217462</isbn>
//         <title lang="en">Being a Dog Is a Full-Time Job</title>
//         <quote>I'd dog paddle the deepest ocean.</quote>
//         <author id="CMS">
//           <?echo "go rocks"?>
//           <name>Charles M Schulz</name>
//           <born>1922-11-26</born>
//           <dead>2000-02-12</dead>
//         </author>
//         <character id="PP">
//           <name>Peppermint Patty</name>
//           <born>1966-08-22</born>
//           <qualification>bold, brash and tomboyish</qualification>
//         </character>
//         <character id="Snoopy">
//           <name>Snoopy</name>
//           <born>1950-10-04</born>
//           <qualification>extroverted beagle</qualification>
//         </character>
//       </book>
//     </library>
//
// The following examples are valid path expressions, and the first
// match has the indicated value:
//
//     /library/book/isbn                               =>  "0836217462"
//     library/*/isbn                                   =>  "0836217462"
//     /library/book/../book/./isbn                     =>  "0836217462"
//     /library/book/character[2]/name                  =>  "Snoopy"
//     /library/book/character[born='1950-10-04']/name  =>  "Snoopy"
//     /library/book//node()[@id='PP']/name             =>  "Peppermint Patty"
//     //book[author/@id='CMS']/title                   =>  "Being a Dog Is a Full-Time Job"},
//     /library/book/preceding::comment()               =>  " Great book. "
//
// To run an expression, compile it, and then apply the compiled path to any
// number of context nodes, from one or more parsed xml documents:
//
//     path := xmlpath.MustCompile("/library/book/isbn")
//     root, err := xmlpath.Parse(file)
//     if err != nil {
//             log.Fatal(err)
//     }
//     if value, ok := path.String(root); ok {
//             fmt.Println("Found:", value)
//     }
//
// To use xmlpath with namespaces, it is required to give the supported set of namespace
// when compiling:
//
//
//    var namespaces = []xmlpath.Namespace {
//        { "s", "http://www.w3.org/2003/05/soap-envelope" },
//        { "a", "http://schemas.xmlsoap.org/ws/2004/08/addressing" },
//    }
//    path, err := xmlpath.CompileWithNamespace("/s:Header/a:To", namespaces)
//    if err != nil {
//            log.Fatal(err)
//    }
//    root, err := xmlpath.Parse(file)
//    if err != nil {
//            log.Fatal(err)
//    }
//    if value, ok := path.String(root); ok {
//            fmt.Println("Found:", value)
//    }
//

package xmlpath
