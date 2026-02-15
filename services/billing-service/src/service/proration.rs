//! Proration calculation for plan upgrades/downgrades.

use chrono::{DateTime, Utc};

/// Proration details for a plan change.
#[derive(Debug, Clone, serde::Serialize)]
pub struct ProrationResult {
    /// Credit for unused time on the old plan (in cents).
    pub credit_cents: i64,
    /// Cost for the remaining time on the new plan (in cents).
    pub charge_cents: i64,
    /// Net amount to charge (charge - credit, can be negative for downgrades).
    pub net_amount_cents: i64,
    /// Number of days remaining in the current period.
    pub days_remaining: i64,
    /// Total days in the current billing period.
    pub total_days: i64,
}

/// Calculate proration for an upgrade or downgrade.
///
/// - `old_price_cents`: current plan price in cents
/// - `new_price_cents`: new plan price in cents
/// - `period_start`: start of current billing period
/// - `period_end`: end of current billing period
/// - `change_date`: when the plan change happens
pub fn calculate_proration(
    old_price_cents: i64,
    new_price_cents: i64,
    period_start: DateTime<Utc>,
    period_end: DateTime<Utc>,
    change_date: DateTime<Utc>,
) -> ProrationResult {
    let total_days = (period_end - period_start).num_days().max(1);
    let days_remaining = (period_end - change_date).num_days().max(0);

    // Daily rate for each plan
    let old_daily_rate = old_price_cents as f64 / total_days as f64;
    let new_daily_rate = new_price_cents as f64 / total_days as f64;

    // Credit for unused days on old plan
    let credit_cents = (old_daily_rate * days_remaining as f64).round() as i64;

    // Charge for remaining days on new plan
    let charge_cents = (new_daily_rate * days_remaining as f64).round() as i64;

    // Net: positive = charge, negative = refund
    let net_amount_cents = charge_cents - credit_cents;

    ProrationResult {
        credit_cents,
        charge_cents,
        net_amount_cents,
        days_remaining,
        total_days,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Duration;

    #[test]
    fn test_upgrade_proration() {
        let now = Utc::now();
        let period_start = now - Duration::days(15);
        let period_end = period_start + Duration::days(30);

        let result = calculate_proration(
            2900,  // $29/mo
            9900,  // $99/mo
            period_start,
            period_end,
            now,
        );

        // 15 days remaining out of 30
        assert_eq!(result.days_remaining, 15);
        assert_eq!(result.total_days, 30);
        assert_eq!(result.credit_cents, 1450); // Half of $29
        assert_eq!(result.charge_cents, 4950); // Half of $99
        assert_eq!(result.net_amount_cents, 3500); // $35 charge
    }

    #[test]
    fn test_downgrade_proration() {
        let now = Utc::now();
        let period_start = now - Duration::days(10);
        let period_end = period_start + Duration::days(30);

        let result = calculate_proration(
            9900,  // $99/mo
            2900,  // $29/mo
            period_start,
            period_end,
            now,
        );

        // 20 days remaining out of 30
        assert_eq!(result.days_remaining, 20);
        assert!(result.net_amount_cents < 0); // Should be a credit
    }

    #[test]
    fn test_same_price_no_proration() {
        let now = Utc::now();
        let period_start = now - Duration::days(15);
        let period_end = period_start + Duration::days(30);

        let result = calculate_proration(
            2900, 2900,
            period_start, period_end, now,
        );

        assert_eq!(result.net_amount_cents, 0);
    }

    #[test]
    fn test_end_of_period_zero_days() {
        let now = Utc::now();
        let period_start = now - Duration::days(30);
        let period_end = now; // Exactly at the end

        let result = calculate_proration(
            2900, 9900,
            period_start, period_end, now,
        );

        assert_eq!(result.days_remaining, 0);
        assert_eq!(result.net_amount_cents, 0);
    }
}
