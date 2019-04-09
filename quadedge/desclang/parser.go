package desclang
/*
The language is pretty simple. It's a line
orinted language, where the first nonspace character
of a line tells us the purpose of the line. Comments
are indicated with the '#' character and runs to the
end of the line. Initially strings will not be supported
but they 

in double quoated strings "" can be used to escape " character.

string !unicode.IsSpace() or '"' IsPrintable() and not '""' or \n till '"'
tags string:stringor "string":"string" or string:"string" or "string":string
float [-+]?[0-9]*\.?[0-9]+([eE][-+]?[0-9]+)?


e float float float float tags
*/


// First the comment
var octothorp = parsect.Atom("#","OCTOTHORP")
var newline = parsect.

// lineString will scan the rest of the line till hit hits a new line character.
func lineString(s Scanner) (ParsecNode, Scanner){

}

