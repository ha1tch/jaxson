// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// Times Apache Jena SHACL validation alone, for scale.py's engine-only
// profile: the shapes and the data are parsed once, then validated N times
// in one JVM after W warm-up runs, and the best run is printed as
//
//   {"ms": 12.3, "conforms": true}
//
// Outside the timer: starting the JVM, writing the data to Turtle, parsing
// it, and parsing the shapes.
//
//   javac -cp "$JENA_LIB/*" -d <dir> JenaTime.java
//   java -cp "<dir>:$JENA_LIB/*" JenaTime shapes.ttl data.ttl [warmup] [runs]
import org.apache.jena.graph.Graph;
import org.apache.jena.riot.RDFDataMgr;
import org.apache.jena.shacl.ShaclValidator;
import org.apache.jena.shacl.Shapes;
import org.apache.jena.shacl.ValidationReport;

public class JenaTime {
    public static void main(String[] a) {
        Graph shapesGraph = RDFDataMgr.loadGraph(a[0]);
        Graph data = RDFDataMgr.loadGraph(a[1]);
        int warmup = a.length > 2 ? Integer.parseInt(a[2]) : 3;
        int runs = a.length > 3 ? Integer.parseInt(a[3]) : 10;
        Shapes shapes = Shapes.parse(shapesGraph);
        double best = Double.MAX_VALUE;
        boolean conforms = false;
        for (int i = 0; i < warmup + runs; i++) {
            long t0 = System.nanoTime();
            ValidationReport r = ShaclValidator.get().validate(shapes, data);
            double ms = (System.nanoTime() - t0) / 1e6;
            conforms = r.conforms();
            if (i >= warmup && ms < best) best = ms;
        }
        System.out.printf(java.util.Locale.ROOT, "{\"ms\": %.4f, \"conforms\": %b}%n", best, conforms);
    }
}
