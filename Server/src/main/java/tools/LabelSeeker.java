package MailClassifier.tools;

import lombok.NonNull;
import org.deeplearning4j.berkeley.Pair;
import org.deeplearning4j.models.embeddings.inmemory.InMemoryLookupTable;
import org.deeplearning4j.models.word2vec.VocabWord;
import org.nd4j.linalg.api.ndarray.INDArray;
import org.nd4j.linalg.ops.transforms.Transforms;

import java.util.ArrayList;
import java.util.List;

public class LabelSeeker {
    private List<String> labelsUsed;
    private InMemoryLookupTable<VocabWord> lookupTable;

    public LabelSeeker(@NonNull List<String> labelsUsed, @NonNull InMemoryLookupTable<VocabWord> lookupTable) {
        if (labelsUsed.isEmpty()) throw new IllegalStateException("You can't have 0 labels used for ParagraphVectors");
        this.lookupTable = lookupTable;
        this.labelsUsed = labelsUsed;
    }

    public List<Pair<String, Double>> getScores(@NonNull INDArray vector) {
        List<Pair<String, Double>> result = new ArrayList<Pair<String, Double>>();
        for (String label: labelsUsed) {
            INDArray vecLabel = lookupTable.vector(label);
            if (vecLabel == null) throw new IllegalStateException("Label '"+ label+"' has no known vector!");

            double sim = Transforms.cosineSim(vector, vecLabel);
            result.add(new Pair<String, Double>(label, sim));
        }
        return result;
    }
}
