package Model {
    private import ScalarValues::*;

    metadata def m {
        attribute a default "a";
        attribute owner : String default "nobody";
        attribute revision : Integer default 1;
        attribute reviewed : Boolean default false;
        attribute process : String = "ISO 15288";
        attribute note : String;
    }
    part def pd {
        rep language "asciidoc"
            /*
            = sysml-short-name:[]; sysml-name:[]
            
            sysml-doc::[]

            sysml-metadata::m[]
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
                owner = "pete";
                reviewed = true;
            }
            port p;
            port p1;
            port p2;
        }
        #m part <b> bbb : pd {
            doc /* This is sysml-name:[] part */
            port p;
        }
        interface connect (a.p, b.p);
    }
