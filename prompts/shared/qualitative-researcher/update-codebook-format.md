<update-codebook-mechanics>
# Merge workflow

When an annotation's ambiguity resolution reveals something the codebook should capture permanently:

1. Update the code definition — add the clarification to inclusion/exclusion criteria or examples
2. Record `user_feedback` on the annotation — capture the researcher's rationale
3. Set `merged: true` on the annotation — signals the resolution has been absorbed into the codebook

The sequence matters: update the codebook first, then mark merged. An annotation marked merged without a corresponding codebook change is misleading.
</update-codebook-mechanics>
