// Fixture for the block macro sysml-catalog::[], which on a view's page
// turns the view's expose + filter into a table: one row per element of the
// exposed tree that carries the filter metadata. The macro takes no target
// or attributes: the root and filter come solely from the page's view
// (the user must not re-state them). The // notes give the expected table.
package Catalog {
    private import ScalarValues::*;
    private import Views::*;
    private import StandardViewDefinitions::*;

    // The filter metadata: catalog rows are the exposed elements that carry a
    // @LogicalProtocolMetadata usage. description gets a presentation label
    // (its rep body) and priority an order, so the columns are ordered and
    // labelled by the presentation facts, not the declared names.
    metadata def LogicalProtocolMetadata {
        // description's catalog column is labelled by this rep body.
        attribute description : String default "none" {
            rep language "asciidoc"
            /* Description */
            // order 2: sorts after priority.
            attribute order default 2;
        }
        // priority's column order is 1, so it comes before Description.
        attribute priority : Integer {
            attribute order default 1;
        }
    }

    // The exposed root: a package of candidate elements. Direct members are
    // tcp, udp, plain and group; the recursive member inner is only reached
    // by a ** expose.
    package Protocols {
        // In the catalog (carries the metadata), documented, so its Name
        // cell links to its page:
        // priority = 1, Description = alpha
        part tcp {
            rep language "asciidoc"
                /* = sysml-name:[] */
            @LogicalProtocolMetadata {
                description = "alpha";
                priority = 1;
            }
        }
        // In the catalog, no bound values:
        // priority = (no value), Description = none (the def default)
        part udp { @LogicalProtocolMetadata; }
        // Not in the catalog: carries no filter metadata.
        part plain;
        // Carries no metadata itself, but owns inner, which is reached by a
        // ** expose only.
        part group {
            // In the deep catalog only:
            // priority = (no value), Description = beta
            part inner { @LogicalProtocolMetadata { description = "beta"; } }
        }
    }

    // An exposed package whose members carry no filter metadata: the line is
    // dropped (nothing to show), as for sysml-members-list.
    package Empty {
        part a;
        part b;
    }

    // Direct-members catalog: expose Protocols::* and a filter.
    // Columns (ordered by their order): Name, priority (order 1),
    //            Description (order 2, the rep body label).
    // Rows: tcp (linked Name, 1, alpha) and udp (none, no value); plain and
    //       group carry no metadata and are dropped.
    view protocolCatalog : GridView {
        rep language "asciidoc"
            /* = sysml-name:[]

               sysml-catalog::[] */
        expose Protocols::*;
        filter @LogicalProtocolMetadata;
    }

    // Recursive catalog: expose Protocols::** reaches inner too.
    // Rows: tcp (1, alpha), udp ((no value), none), inner ((no value), beta).
    view protocolCatalogDeep : GridView {
        rep language "asciidoc"
            /* = sysml-name:[]

               sysml-catalog::[] */
        expose Protocols::**;
        filter @LogicalProtocolMetadata;
    }

    // An empty catalog: nothing exposed carries the filter metadata, so the
    // macro line is dropped.
    view emptyCatalog : GridView {
        rep language "asciidoc"
            /* = sysml-name:[]

               sysml-catalog::[] */
        expose Empty::*;
        filter @LogicalProtocolMetadata;
    }

    // A view with no expose: the macro fails.
    view noExpose : GridView {
        rep language "asciidoc"
            /* = sysml-name:[]

               sysml-catalog::[] */
        filter @LogicalProtocolMetadata;
    }

    // A view with no filter: the macro fails.
    view noFilter : GridView {
        rep language "asciidoc"
            /* = sysml-name:[]

               sysml-catalog::[] */
        expose Protocols::*;
    }

    // Not a view: the macro fails.
    part notAView {
        rep language "asciidoc"
            /* = sysml-name:[]

               sysml-catalog::[] */
    }
}