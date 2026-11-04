package Model {
    metadata def m {
        attribute a default "a";
    }
    part def pd {
        rep language "asciidoc"
            /*Textual representation*/
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
