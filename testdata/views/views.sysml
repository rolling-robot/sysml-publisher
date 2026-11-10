// Fixture for the block macro sysml-view::[<view>][], replaced by the named
// view as OpenSysML renders it: a picture for a diagram, a table for a table
// view. The // notes give what each page shows.
package V {
    private import Views::*;
    private import StandardViewDefinitions::*;
    port def P;
    part def Box { port p : P; port q : P; }

    part sys {
        rep language "asciidoc"
            /* = System

               sysml-view::interfaces[]

               .Parts
               sysml-view::partsTable[] */
        part x : Box;
        part y : Box;
        // sys's page: the SVG of interfaces (x and y, connected by i), then
        // the table of partsTable, titled Parts.
        interface i connect x.p to y.p;

        // Named relative to sys's page: an SVG.
        view interfaces {
            expose sys;
            render asInterconnectionDiagram;
        }
        // Not drawn by the test renderer: the macro fails.
        view unrendered {
            expose sys;
            render asInterconnectionDiagram;
        }
    }

    // Shows its own view without a target: a table of sys's members (x, y,
    // i, interfaces, unrendered).
    view partsTable {
        rep language "asciidoc"
            /* sysml-view::[] */
        expose sys::*;
        render asElementTable;
    }

    // Not a view: the macro fails.
    part notAView {
        rep language "asciidoc"
            /* sysml-view::sys[] */
    }
}
