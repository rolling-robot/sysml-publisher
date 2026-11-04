package Model {
    metadata def m {
        attribute a default "a";
    }
    part def pd {
        rep language "asciidoc"
            /*
            = sysml-short-name:[]; sysml-name:[]
            
            sysml-doc::[]

            */
    }
        part <a> aaaa : pd {
            doc /*
            Documentation for part sysml-name:[]

            Owned elements:

            sysml-members::[]
            */
            @m {
                a := "aa";
            }
            port p;
            port p1;
            port p2;
        }
        part <b> bbb : pd {
            doc /* This is sysml-name:[] part */
            port p;
        }
        interface connect (a.p, b.p);
    }
