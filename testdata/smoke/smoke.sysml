package Model {
    metadata def m {
        attribute a default "a";
    }
    part def pd {
        rep language "asciidoc"
            /*
            = Element: sysml-name:[]

            Textual representation

            sysml-name:[]
            */
    }
    part <a> aaaa : pd {
        doc /*Documentation*/
        @m {
            a := "aa";
        }
        port p;
    }
    part <b> bbb : pd {
        port p;
    }
    interface connect (a.p, b.p);
}
