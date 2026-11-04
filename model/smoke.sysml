package Model {
    metadata def m {
        attribute a default "a";
    }
    part def pd {
        rep language "asciidoc"
            /*
            = sysml-name:[]
            
            sysml-doc::[]

            sysml-name:[]
            */
    }
        part <a> aaaa : pd {
            doc /*
            Documentation for part a
            */
            @m {
                a := "aa";
            }
            port p;
        }
        part <b> bbb : pd {
            doc /* This is sysml-name:[] part */
            port p;
        }
        interface connect (a.p, b.p);
    }
