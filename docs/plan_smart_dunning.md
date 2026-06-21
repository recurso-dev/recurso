# Implementation Plan: Smart Dunning (RL-Powered) 🧠

## Overview
This plan outlines the steps to implement a Contextual Multi-Armed Bandit (MAB) system for optimizing payment retries. The system will use an Epsilon-Greedy strategy to explore and exploit different retry intervals.

## Phase 1: Domain & Data Modeling
1.  **Define RL Entities**: Create `DunningAction` (retry intervals) and `DunningReward` (success/fail) models.
2.  **Schema Migration**: Create `dunning_history` and `dunning_weights` tables to persist the model state.

## Phase 2: Core Bandit Implementation
1.  **Service Logic**: Implement the `SmartRetryService` with:
    - `SelectAction(context)`: Chooses an interval (Exploration vs. Exploitation).
    - `UpdateWeights(action, reward)`: Updates the probability distribution based on outcome.
2.  **Context Extraction**: Logic to map an `Invoice` + `FailureReason` to a feature vector.

## Phase 3: Repository & Persistence
1.  **Dunning Repository**: Implement PostgreSQL methods to fetch and update weights based on context keys (e.g., `INR:insufficient_funds`).

## Phase 4: Integration
1.  **Retry Worker Update**: Modify the existing `worker.RetryWorker` to call the `SmartRetryService` instead of using static intervals.
2.  **Outcome Tracking**: Ensure that when a payment succeeds or fails, the `SmartRetryService` is notified to record the reward.

## Phase 5: Verification & Simulation
1.  **Simulated Environment**: Create a test that runs 1000 iterations and asserts that the Bandit converges on the "best" action for a given mock context.

## Risks & Mitigations
- **Cold Start**: The model will start with equal weights for all actions (random exploration) before converging.
- **Latency**: Use in-memory caching for weights to ensure retry interval calculation is sub-millisecond.
- **Drift**: Implement a minimum exploration rate (epsilon) to ensure the model adapts if gateway behavior changes.
