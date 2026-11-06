package M {
    part def <P> pd {
        doc /* pd documentation. */
        rep language "adoc"
            /* Name: sysml-name:[]. */
    }
    part 'a<b' : pd {
        doc /* The *a<b* part. */
        doc /* Second doc. */
    }
    part : pd;
    // Appended after the unnamed part, so it stays M::@2. For sysml-name
    // with a target: an element without a page whose quoted name has a
    // space, and a name two elements share (an ambiguous target).
    part def Other {
        doc /* Other's documentation. */
        attribute 'top speed';
    }
    part def A { attribute x; }
    part def B { attribute x; }
}
