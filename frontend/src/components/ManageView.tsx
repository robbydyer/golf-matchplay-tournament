import { useState } from 'react';
import { Tournament } from '../types';
import * as api from '../api/client';

interface Props {
  tournament: Tournament;
  onUpdate: () => void;
}

const DEFAULT_HEADER = '#1C4932';
const DEFAULT_BG = '#f5f5f0';

export default function ManageView({ tournament, onUpdate }: Props) {
  const [headerColor, setHeaderColor] = useState(tournament.headerColor || DEFAULT_HEADER);
  const [bgColor, setBgColor] = useState(tournament.bgColor || DEFAULT_BG);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      await api.updateTournament(tournament.id, { headerColor, bgColor });
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = async () => {
    setHeaderColor(DEFAULT_HEADER);
    setBgColor(DEFAULT_BG);
  };

  return (
    <div className="manage-view">
      {error && <div className="error">{error}</div>}

      <div className="card manage-card">
        <h3>Appearance</h3>

        <div className="manage-colors">
          <div className="form-group">
            <label>Header Bar Color</label>
            <div className="team-color-input">
              <input
                type="color"
                value={headerColor}
                onChange={(e) => setHeaderColor(e.target.value)}
              />
              <span>{headerColor}</span>
            </div>
          </div>

          <div className="form-group">
            <label>Background Color</label>
            <div className="team-color-input">
              <input
                type="color"
                value={bgColor}
                onChange={(e) => setBgColor(e.target.value)}
              />
              <span>{bgColor}</span>
            </div>
          </div>
        </div>

        <div className="form-actions">
          <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
          <button className="btn" onClick={handleReset}>
            Reset to Defaults
          </button>
        </div>
      </div>
    </div>
  );
}
