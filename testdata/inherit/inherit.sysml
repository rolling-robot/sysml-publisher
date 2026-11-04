package Inherit {
    part def pd {
        rep language "adoc"
            /* pd text */
    }
    part def pd2 :> pd {
        rep language "adoc"
            /* pd2 text */
    }
    part x : pd2 {
        rep language "adoc"
            /* x text */
    }
    part y :> x;

    part def A {
        rep language "adoc"
            /* A text */
    }
    part def B :> A;
    part def C :> A;
    part def D :> B, C;
}
