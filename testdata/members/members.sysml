package M {
    metadata def tag;
    part def other;
    part def pd {
        doc /* pd documentation. */
        rep language "adoc"
            /* Members: */
        attribute mass;
        port 'a*b*+++c';
        part sub {
            rep language "adoc"
                /* sub has a page. */
        }
        part : other;
    }
    part leaf : pd {
        @tag;
        port q;
    }
}
